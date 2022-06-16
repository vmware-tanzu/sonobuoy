/*
Copyright the Sonobuoy contributors 2019

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/discovery"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"gopkg.in/yaml.v2"
)

const (
	// resultModeReport prints a human-readable summary of the results to stdout.
	resultModeReport = "report"

	// resultModeDetailed will dump each leaf node (e.g. test) as a json object. If the results
	// are just references to files (like systemd-logs) then it will print the file for each
	// leaf node, prefixed with the path.
	resultModeDetailed = "detailed"

	// resultModeDump will just copy the post-processed yaml file to stdout.
	resultModeDump = "dump"

	//resultModeReadable will copy the post-processed yaml file to stdout and replace \n and \t with new lines and tabs respectively.
	resultModeReadable = "readable"

	windowsSeperator = `\`

	//Name of the "fake" plugin used to enable printing the health summary.
	//This name needs to be reserved to avoid conflicts with a plugin with the same name
	clusterHealthSummaryPluginName = "sonobuoy"
)

type resultsInput struct {
	archive    string
	plugin     string
	mode       string
	node       string
	skipPrefix bool
}

func NewCmdResults() *cobra.Command {
	data := resultsInput{}
	cmd := &cobra.Command{
		Use:   "results archive.tar.gz",
		Short: "Inspect plugin results.",
		Run: func(cmd *cobra.Command, args []string) {
			data.archive = args[0]
			if err := result(data); err != nil {
				errlog.LogError(errors.Wrapf(err, "could not process archive: %v", args[0]))
				os.Exit(1)
			}
		},
		Args: cobra.ExactArgs(1),
	}

	cmd.Flags().StringVarP(
		&data.plugin, "plugin", "p", "",
		"Which plugin to show results for. Defaults to printing them all.",
	)
	cmd.Flags().StringVarP(
		&data.mode, "mode", "m", resultModeReport,
		`Modifies the format of the output. Valid options are report, detailed, readable, or dump.`,
	)
	cmd.Flags().StringVarP(
		&data.node, "node", "n", "",
		`Traverse results starting at the node with the given name. Defaults to the real root.`,
	)
	cmd.Flags().BoolVarP(
		&data.skipPrefix, "skip-prefix", "s", false,
		`When printing items linking to files, only print the file contents.`,
	)

	return cmd
}

// getReader returns a *results.Reader along with a cleanup function to close the
// underlying readers. The cleanup function is guaranteed to never be nil.
func getReader(filepath string) (*results.Reader, func(), error) {
	fi, err := os.Stat(filepath)
	if err != nil {
		return nil, func() {}, err
	}
	if fi.IsDir() {
		return results.NewReaderFromDir(filepath), func() {}, nil
	}
	f, err := os.Open(filepath)
	if err != nil {
		return nil, func() {}, errors.Wrapf(err, "could not open sonobuoy archive: %v", filepath)
	}

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return nil, func() { f.Close() }, errors.Wrap(err, "could not make a gzip reader")
	}

	r := results.NewReaderWithVersion(gzr, results.VersionTen)
	return r, func() { gzr.Close(); f.Close() }, nil
}

// result takes the resultsInput and tries to print the requested infromation from the archive.
// If there is an error printing any individual plugin, only the last error is printed and all plugins
// continue to be processed.
func result(input resultsInput) error {
	r, cleanup, err := getReader(input.archive)
	defer cleanup()
	if err != nil {
		return err
	}

	// Report on all plugins or the specified one.
	plugins := []string{input.plugin}
	if len(input.plugin) == 0 {
		plugins, err = getPluginList(r)
		if err != nil {
			return errors.Wrapf(err, "unable to determine plugins to report on")
		}
		if len(plugins) == 0 {
			return fmt.Errorf("no plugins specified by either the --plugin flag or tarball metadata")
		}
		//Add clusterHealthSummaryPluginName only after we verified there is at least another plugin
		plugins = append(plugins, clusterHealthSummaryPluginName)
	}

	var lastErr error
	for i, plugin := range plugins {
		input.plugin = plugin

		// Load file with a new reader since we can't assume this reader has rewind
		// capabilities.
		r, cleanup, err := getReader(input.archive)
		defer cleanup()
		if err != nil {
			lastErr = err
		}

		//bypass if this plugin is called clusterHealthSummaryPluginName
		if input.plugin == clusterHealthSummaryPluginName {
			err = printHealthSummary(input, r)
			if err != nil {
				lastErr = err
			}
			continue
		}

		err = printSinglePlugin(input, r)
		if err != nil {
			lastErr = err
		}

		// Seperator line, but don't print a needless one at the end.
		if i+1 < len(plugins) {
			fmt.Println()
		}
	}

	return lastErr
}

// printHealthSummary pretends to work like printSinglePlugin
// but for a "fake" plugin that prints health information
func printHealthSummary(input resultsInput, r *results.Reader) error {
	var err error

	//For detailed view we can just dump the contents of the clusterHealthSummaryPluginName file
	if input.mode == resultModeDetailed {
		reader, err := r.FileReader(results.ClusterHealthFilePath())
		if err != nil {
			return errors.Wrapf(err, "failed to get health summary results reader from file '%s'", results.ClusterHealthFilePath())
		}
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			return errors.Wrapf(err, "failed to copy health summary results from file '%s'", results.ClusterHealthFilePath())
		}
		return nil
	}

	// For summary and dump views, get the item as an object to iterate over.
	clusterHealthSummary := discovery.ClusterSummary{}

	err = r.WalkFiles(func(path string, info os.FileInfo, err error) error {
		return results.ExtractFileIntoStruct(results.ClusterHealthFilePath(), path, info, &clusterHealthSummary)
	})
	if err != nil {
		return err
	}

	var data []byte

	switch input.mode {
	case resultModeDump:
		data, err = yaml.Marshal(clusterHealthSummary)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case resultModeReadable:
		data, err = yaml.Marshal(clusterHealthSummary)
		if err != nil {
			return err
		}
		str := string(data)
		str = strings.ReplaceAll(str, `\n`, "\n")
		str = strings.ReplaceAll(str, `\t`, "	")
		fmt.Println(str)
	default:
		err = printClusterHealthResultsSummary(clusterHealthSummary)
		if err != nil {
			return err
		}
	}
	return nil
}

type humanReadableWriter struct {
	w io.Writer
}

func (hw *humanReadableWriter) Write(b []byte) (int, error) {
	newb := bytes.Replace(b, []byte(`\n`), []byte("\n"), -1)
	newb = bytes.Replace(newb, []byte(`\t`), []byte("\t"), -1)
	_, err := hw.w.Write(newb)
	return len(b), err
}

func printSinglePlugin(input resultsInput, r *results.Reader) error {
	// If we want to dump the whole file, don't decode to an Item object first.
	if input.mode == resultModeDump {
		fReader, err := r.PluginResultsReader(input.plugin)
		if err != nil {
			return errors.Wrapf(err, "failed to get results reader for plugin %v", input.plugin)
		}
		_, err = io.Copy(os.Stdout, fReader)
		return err
	} else if input.mode == resultModeReadable {
		fReader, err := r.PluginResultsReader(input.plugin)
		if err != nil {
			return errors.Wrapf(err, "failed to get results reader for plugin %v", input.plugin)
		}
		writer := &humanReadableWriter{os.Stdout}
		_, err = io.Copy(writer, fReader)
		if err != nil {
			return errors.Wrapf(err, "failed to copy data for plugin %v", input.plugin)
		}
		return err
	}

	// For summary and detailed views, get the item as an object to iterate over.
	obj, err := r.PluginResultsItem(input.plugin)
	if err != nil {
		return err
	}

	obj = obj.GetSubTreeByName(input.node)
	if obj == nil {
		return fmt.Errorf("node named %q not found", input.node)
	}

	switch input.mode {
	case resultModeDetailed:
		return printResultsDetails([]string{}, obj, input)
	default:
		return printResultsSummary(obj)
	}
}

func getPluginList(r *results.Reader) ([]string, error) {
	runInfo := discovery.RunInfo{}
	err := r.WalkFiles(func(path string, info os.FileInfo, err error) error {
		return results.ExtractFileIntoStruct(r.RunInfoFile(), path, info, &runInfo)
	})

	return runInfo.LoadedPlugins, errors.Wrap(err, "finding plugin list")
}

func printResultsDetails(treePath []string, o *results.Item, input resultsInput) error {
	if o == nil {
		return nil
	}

	if len(o.Items) > 0 {
		treePath = append(treePath, o.Name)
		for _, v := range o.Items {
			if err := printResultsDetails(treePath, &v, input); err != nil {
				return err
			}
		}
		return nil
	}

	leafFile := getFileFromMeta(o.Metadata)
	if leafFile == "" {
		// Print each leaf node as a json object. Add the path as a metadata field for access by the end user.
		if o.Metadata == nil {
			o.Metadata = map[string]string{}
		}
		o.Metadata["path"] = strings.Join(treePath, "|")
		b, err := json.Marshal(o)
		if err != nil {
			return errors.Wrap(err, "marshalling item to json")
		}
		fmt.Println(string(b))
	} else {
		// Load file with a new reader since we can't assume this reader has rewind
		// capabilities.
		r, cleanup, err := getReader(input.archive)
		defer cleanup()
		if err != nil {
			return errors.Wrapf(err, "reading archive to get file %v", leafFile)
		}
		resultFile := path.Join(results.PluginsDir, input.plugin, leafFile)
		filereader, err := r.FileReader(resultFile)
		if err != nil {
			return err
		}

		if input.skipPrefix {
			_, err = io.Copy(os.Stdout, filereader)
			return err
		} else {
			// When printing items like this we want the name of the node in
			// the prefix. In the "junit" version, we do not, since the name is
			// already visible on the object.
			treePath = append(treePath, o.Name)
			fmt.Printf("%v ", strings.Join(treePath, "|"))
			_, err = io.Copy(os.Stdout, filereader)
			return err
		}
	}

	return nil
}

// sortErrors takes a discovery.LogSummary, which is a map[string]map[string]int
// which is a map[errorName]map[fileName]errorCount
// and returns a map[string]i[]string that is a map[errorName] where the value
// is a list of file names ordered by the errorCount (descending)
func sortErrors(errorSummary discovery.LogSummary) map[string][]string {
	result := make(map[string][]string)
	for errorName, hitCounter := range errorSummary {
		sortedFileNamesList := make([]string, 0)
		for fileName := range hitCounter {
			sortedFileNamesList = append(sortedFileNamesList, fileName)
		}
		//Sort in descending order,
		//And use the values in hitCounter for the sorting
		isMore := func(i, j int) bool {
			valueI := hitCounter[sortedFileNamesList[i]]
			valueJ := hitCounter[sortedFileNamesList[j]]
			return valueI > valueJ
		}
		sort.Slice(sortedFileNamesList, isMore)
		result[errorName] = sortedFileNamesList
	}
	return result
}

// filterAndSortHealthInfoDetails takes a copy of a slice of HealthInfoDetails,
// discards the ones that are healthy,
// then sorts the remaining entries,
// and finally sorts them by namespace and name
func filterAndSortHealthInfoDetails(details []discovery.HealthInfoDetails) []discovery.HealthInfoDetails {
	result := make([]discovery.HealthInfoDetails, len(details))
	var idx int
	for _, detail := range details {
		if !detail.Healthy {
			result[idx] = detail
			idx++
		}
	}
	result = result[:idx]
	isLess := func(i, j int) bool {
		if result[i].Namespace == result[j].Namespace {
			return result[i].Name < result[j].Name
		} else {
			return result[i].Namespace < result[j].Namespace
		}
	}
	sort.Slice(result, isLess)
	return result
}

// printClusterHealthResultsSummary prints the summary of the "fake" plugin for health summary,
// tryingf to emulate the format of printResultsSummary
func printClusterHealthResultsSummary(summary discovery.ClusterSummary) error {
	fmt.Println("Run Details:")
	fmt.Printf("API Server version: %s\n", summary.APIVersion)

	fmt.Printf("Node health: %d/%d", summary.NodeHealth.Healthy, summary.NodeHealth.Total)
	//Print the percentage only if Total is not 0 to avoid division by zero errors
	if summary.NodeHealth.Total != 0 {
		fmt.Printf(" (%d%%)", 100*summary.NodeHealth.Healthy/summary.NodeHealth.Total)
	}
	fmt.Println()
	//Details of the failed pods. Checking the slice length to avoid trusting the Total
	if len(summary.NodeHealth.Details) > 0 && summary.NodeHealth.Healthy < summary.NodeHealth.Total {
		fmt.Println("Details for failed nodes:")
		nodes := filterAndSortHealthInfoDetails(summary.NodeHealth.Details)
		for _, node := range nodes {
			fmt.Printf("%s Ready:%s: %s: %s\n", node.Name, node.Ready, node.Reason, node.Message)
		}
		fmt.Println()
	}

	//It might be nice to group pods by namespace.
	//Also here, use len instead of trusting Total
	if len(summary.PodHealth.Details) > 0 {
		fmt.Printf("Pods health: %d/%d", summary.PodHealth.Healthy, summary.PodHealth.Total)
		//Print the percentage only if Total is not 0 to avoid division by zero errors
		if summary.PodHealth.Total != 0 {
			fmt.Printf(" (%d%%)", 100*summary.PodHealth.Healthy/summary.PodHealth.Total)
		}
		fmt.Println()
		if summary.PodHealth.Healthy < summary.PodHealth.Total {
			fmt.Println("Details for failed pods:")
			pods := filterAndSortHealthInfoDetails(summary.PodHealth.Details)
			//And then print them, sorted by namespace
			for _, pod := range pods {
				fmt.Printf("%s/%s Ready:%s: %s: %s\n", pod.Namespace, pod.Name, pod.Ready, pod.Reason, pod.Message)
			}
		}
	}

	if len(summary.ErrorInfo) > 0 {
		fmt.Println("Errors detected in files:")
		sortedFileNames := sortErrors(summary.ErrorInfo)
		for errorType := range summary.ErrorInfo {
			//Get the first item in the list of sorted file names and get the value for that file name
			maxValue := summary.ErrorInfo[errorType][sortedFileNames[errorType][0]]
			//Calculate the width of the string representation of the maxValue
			maxWidth := len(fmt.Sprintf("%d", maxValue))
			fmt.Printf("%s:\n", errorType)

			for _, fileName := range sortedFileNames[errorType] {
				fmt.Printf("%[1]*[2]d %[3]s\n", maxWidth, summary.ErrorInfo[errorType][fileName], fileName)
			}
		}
	}

	return nil
}

func printResultsSummary(o *results.Item) error {
	statusCounts := map[string]int{}
	var failedList []string

	statusCounts, failedList = walkForSummary(o, statusCounts, failedList)

	total := 0
	for _, v := range statusCounts {
		total += v
	}

	fmt.Println("Plugin:", o.Name)
	fmt.Println("Status:", o.Status)
	fmt.Println("Total:", total)

	// We want to print the built-in status type results first before printing any custom statuses, so print first then delete.
	fmt.Println("Passed:", statusCounts[results.StatusPassed])
	fmt.Println("Failed:", statusCounts[results.StatusFailed]+statusCounts[results.StatusTimeout])
	fmt.Println("Skipped:", statusCounts[results.StatusSkipped])

	delete(statusCounts, results.StatusPassed)
	delete(statusCounts, results.StatusFailed)
	delete(statusCounts, results.StatusTimeout)
	delete(statusCounts, results.StatusSkipped)

	// We want the custom statuses to always be printed in order so sort them before proceeding
	keys := []string{}
	for k := range statusCounts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%v: %v\n", k, statusCounts[k])
	}

	if len(failedList) > 0 {
		fmt.Print("\nFailed tests:\n")
		fmt.Print(strings.Join(failedList, "\n"))
		fmt.Println()
	}

	return nil
}

func walkForSummary(result *results.Item, statusCounts map[string]int, failList []string) (map[string]int, []string) {
	if len(result.Items) > 0 {
		for _, item := range result.Items {
			statusCounts, failList = walkForSummary(&item, statusCounts, failList)
		}
		return statusCounts, failList
	}

	statusCounts[result.Status]++

	if result.Status == results.StatusFailed || result.Status == results.StatusTimeout {
		failList = append(failList, result.Name)
	}

	return statusCounts, failList
}

// getFileFromMeta pulls the file out of the given metadata but also
// converts it to a slash-based-seperator since that is what is internal
// to the tar file. The metadata is written by the node and so may use
// Windows seperators.
func getFileFromMeta(m map[string]string) string {
	if m == nil {
		return ""
	}
	return toSlash(m["file"])
}

// toSlash is a (for our purpose) an improved version of filepath.ToSlash which ignores the
// current OS seperator and simply converts all windows `\` to `/`.
func toSlash(path string) string {
	return strings.ReplaceAll(path, string(windowsSeperator), "/")
}

// stringsToRegexp just makes a regexp out of the string array that will match any of the given values.
func stringsToRegexp(testCases []string) string {
	testNames := make([]string, len(testCases))
	for i, tc := range testCases {
		testNames[i] = regexp.QuoteMeta(tc)
	}
	return strings.Join(testNames, "|")
}

func failedTestsFromTar(tarballPath, plugin string) ([]string, error) {
	r, cleanup, err := getReader(tarballPath)
	defer cleanup()
	if err != nil {
		return nil, err
	}

	obj, err := r.PluginResultsItem(plugin)
	if err != nil {
		return nil, err
	}

	statusCounts := map[string]int{}
	var failedList []string
	_, failedList = walkForSummary(obj, statusCounts, failedList)
	return failedList, nil
}
