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
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	"github.com/vmware-tanzu/sonobuoy/pkg/discovery"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
		`Modifies the format of the output. Valid options are report, detailed, or dump.`,
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
	}

	var lastErr error
	for i, plugin := range plugins {
		input.plugin = plugin

		// Load file with a new reader since we can't assume this reader has rewind
		// capabilities.
		r, cleanup, err := getReader(input.archive)
		defer cleanup()

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

func printSinglePlugin(input resultsInput, r *results.Reader) error {
	// If we want to dump the whole file, don't decode to an Item object first.
	if input.mode == resultModeDump {
		fReader, err := r.PluginResultsReader(input.plugin)
		if err != nil {
			return errors.Wrapf(err, "failed to get results reader for plugin %v", input.plugin)
		}
		_, err = io.Copy(os.Stdout, fReader)
		return err
	}

	// For summary and detailed views, get the item as an object to iterate over.
	obj, err := r.PluginResultsItem(input.plugin)
	if err != nil {
		return err
	}

	obj = getItemInTree(obj, input.node)
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

func getItemInTree(i *results.Item, root string) *results.Item {
	if i == nil {
		return nil
	}

	if root == "" || i.Name == root {
		return i
	}

	if len(i.Items) > 0 {
		for _, v := range i.Items {
			subItem := getItemInTree(&v, root)
			if subItem != nil {
				return subItem
			}
		}
	}

	return nil
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

func printResultsSummary(o *results.Item) error {
	p, f, s, failList := walkForSummary(o, 0, 0, 0, []string{})

	fmt.Printf(`Plugin: %v
Status: %v
Total: %v
Passed: %v
Failed: %v
Skipped: %v
`, o.Name, o.Status, p+f+s, p, f, s)

	if len(failList) > 0 {
		fmt.Print("\nFailed tests:\n")
		fmt.Print(strings.Join(failList, "\n"))
		fmt.Println()
	}

	return nil
}

func walkForSummary(o *results.Item, p, f, s int, failList []string) (numPassed, numFailed, numSkipped int, failed []string) {
	if len(o.Items) > 0 {
		for _, v := range o.Items {
			p, f, s, failList = walkForSummary(&v, p, f, s, failList)
		}
		return p, f, s, failList
	}

	switch o.Status {
	case results.StatusPassed:
		p++
	case results.StatusFailed:
		f++
		failList = append(failList, o.Name)
	case results.StatusSkipped:
		s++
	}

	return p, f, s, failList
}

func getFileFromMeta(m map[string]string) string {
	if m == nil {
		return ""
	}
	return m["file"]
}
