package app

import (
	"bufio"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/image"
	"k8s.io/client-go/discovery"
)

const (
	e2ePrintModeTagsOnly     = "tags"
	e2ePrintModeTagsAndCount = "tagCounts"
	e2ePrintModeTests        = "tests"

	e2eInputOnline  = "online"
	e2eInputOffline = "offline"
	e2eInputStdin   = "-"
)

//var defaultBaseURL = "https://raw.githubusercontent.com/vmware-tanzu/sonobuoy/main/cmd/sonobuoy/app/e2e/testLists"
var defaultBaseURL = "https://raw.githubusercontent.com/vmware-tanzu/sonobuoy/main/cmd/sonobuoy/app/e2e/customTestLists"

//go:embed e2e/testLists/*
var e2eTestListFS embed.FS

type e2eFlags struct {
	focus string
	skip  string

	input           string
	k8sVersion      image.ConformanceImageVersion
	kubecfg         Kubeconfig
	resolvedVersion string
	baseURL         string

	mode string
}

// testListStubError is the type of error returned when we try and read tests from a file/url
// but it references another version. Rather than handle it in the reader method, we bubble this
// error up so that we have more context on the initial user request.
type testListStubError struct {
	ReferenceVersion string
	AddTests         []string
	RemoveTests      []string
}

func (e *testListStubError) Error() string {
	return fmt.Sprintf("test list stub: attempted to load tests but the list references version %v", e.ReferenceVersion)
}

func NewCmdE2E() *cobra.Command {
	f := e2eFlags{}
	cmd := &cobra.Command{
		Use:   "e2e",
		Short: "Generates a list of all tests and tags in that tests",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// In some configurations, the kube client isn't actually needed for correct executation
			// Therefore, delay reporting the error until we're sure we need the client
			kubeclient, kubeError := getClient(&f.kubecfg)

			var discoveryClient discovery.ServerVersionInterface
			if kubeclient != nil {
				discoveryClient = kubeclient.DiscoveryClient
			}

			// `auto` k8s version needs resolution as well as any static plugins which use the
			// variable SONOBUOY_K8S_VERSION. Just check for it all by default but allow skipping
			// errors/resolution via flag.
			_, k8sVersion, err := f.k8sVersion.Get(discoveryClient, "")
			if err != nil {
				if errors.Cause(err) == image.ErrImageVersionNoClient {
					return errors.Wrap(err, kubeError.Error())
				}
				return err
			}
			f.resolvedVersion = k8sVersion
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return e2eSonobuoyRun(&f)
		},
		Args: cobra.ExactArgs(0),
	}
	AddKubeconfigFlag(&f.kubecfg, cmd.Flags())
	cmd.Flags().StringVarP(&f.mode, "mode", "m", "tests", "Print mode. Can be one of [tags, tagCounts, tests]")
	cmd.Flags().StringVarP(&f.focus, "focus", "f", "", "Return tests which match this regular expression")
	cmd.Flags().StringVarP(&f.skip, "skip", "s", "", "Do not return tests which match this regular expression")
	cmd.Flags().StringVarP(&f.input, "input", "i", "online", "Determines the source of the test lists. Can be [online, offline, -]. If '-' is set, tests will be read from stdin.")

	testListURL := os.Getenv("DEFAULT_BASE_URL")
	if testListURL != "" {
		fmt.Println("DEFAULT_BASE_URL env set. Using DEFAULT_BASE_URL path to read testLists")
		defaultBaseURL = testListURL
	}
	fmt.Printf("defaultBaseURL: %+v\n", defaultBaseURL)
	
	// Hidden flag to override base URL if we have issues. Prevents older releases from being broken due to changing URL value.
	cmd.Flags().StringVar(&f.baseURL, "url", defaultBaseURL, "The base URL in github to find the test lists for each version.")
	cmd.Flags().MarkHidden("url")

	help := "Use default E2E image, but override the version. "
	help += "Default is 'auto', which will be set to your cluster's version if detected, erroring otherwise. "
	help += "'ignore' will try version resolution but ignore errors. "
	help += "'latest' will find the latest dev image/version upstream."
	cmd.Flags().Var(&f.k8sVersion, "version", help)

	return cmd
}

func e2eSonobuoyRun(e *e2eFlags) error {
	fmt.Printf("In e2eSonobuoyRun function. e2eFlags: %+v", e)
	testList, err := getTests(e.input, e.baseURL, e.resolvedVersion)
	if err != nil {
		return err
	}
	if len(testList) == 0 {
		return fmt.Errorf("no tests found with given options")
	}

	var f, s *regexp.Regexp
	if len(e.focus) > 0 {
		f, err = regexp.Compile(e.focus)
		if err != nil {
			return errors.Wrapf(err, "failed to compile focus value")
		}
	}
	if len(e.skip) > 0 {
		s, err = regexp.Compile(e.skip)
		if err != nil {
			return errors.Wrapf(err, "failed to compile focus value")
		}
	}
	testList = filterTests(testList, f, s)
	printTestList(os.Stdout, e.mode, testList)
	return nil
}

func printTestList(w io.Writer, mode string, list []string) {
	switch mode {
	case e2ePrintModeTagsOnly:
		logrus.Tracef("Printing mode tags only")
		tagMap := tagCountsFromList(list)
		for _, v := range sortedKeys(tagMap) {
			fmt.Fprintln(w, v)
		}
	case e2ePrintModeTagsAndCount:
		logrus.Tracef("Printing mode tags and counts")
		tagMap := tagCountsFromList(list)
		for _, v := range sortedKeys(tagMap) {
			fmt.Fprintf(w, "%v:%v\n", v, tagMap[v])
		}
	case e2ePrintModeTests:
		logrus.Tracef("Printing mode set to print full test names")
		fmt.Fprintln(w, strings.Join(list, "\n"))
	default:
		logrus.Tracef("Unknown printing mode; printing full test names")
		fmt.Fprintln(w, strings.Join(list, "\n"))
	}
}

func sortedKeys(input map[string]int) []string {
	keys := []string{}
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func tagCountsFromList(list []string) map[string]int {
	resultMap := map[string]int{}
	r := regexp.MustCompile(`\[.*?\]`)
	for _, v := range list {
		tags := r.FindAllString(v, -1)
		for _, t := range tags {
			resultMap[t]++
		}
	}
	return resultMap
}

func filterTests(list []string, focus, skip *regexp.Regexp) []string {
	returnList := []string{}
	for _, v := range list {
		// Nil focus implies match anything. Nil skip implies do NOT skip anything.
		if (focus == nil || focus.MatchString(v)) && (skip == nil || !skip.MatchString(v)) {
			returnList = append(returnList, v)
		}
	}
	return returnList
}

func getTests(input, baseURL, version string) ([]string, error) {
	fmt.Printf("In getTests function. input: %+v, baseURL: %+v, version:%+v", input, baseURL, version)
	var tests []string
	var err error

	switch input {
	case e2eInputOnline:
		tests, err = getTestsOnline(baseURL, version)
	case e2eInputOffline:
		tests, err = getTestsOffline(version)
	case e2eInputStdin:
		tests, err = getTestsStdin()
	default:
		err = fmt.Errorf("unknown input option set: %q, expected one of [%v, %v, %v]", input, e2eInputOnline, e2eInputOffline, e2eInputStdin)
	}
	var stubErr *testListStubError
	if errors.As(err, &stubErr) {
		logrus.Tracef("Attempted to load tests for version %v but it referenced version %v. Loading that version with other options the same.", version, stubErr.ReferenceVersion)

		// Recurse with all the same options. Return on errors (they'll be non-testListStubError).
		tests, err = getTests(input, baseURL, stubErr.ReferenceVersion)
		if err != nil {
			return nil, err
		}
		tests = mergeResults(tests, stubErr.AddTests, stubErr.RemoveTests)
	}
	return tests, err
}

func mergeResults(tests, addTests, removeTests []string) []string {
	result := append(tests, addTests...)
	sort.Strings(result)
	for _, removeVal := range removeTests {
		i := sort.SearchStrings(result, removeVal)
		if result[i] == removeVal {
			result = append(result[:i], result[i+1:]...)
		}
	}
	return result
}

func getTestsStdin() ([]string, error) {
	return testsFromReader(os.Stdin)
}

func getTestsOffline(version string) ([]string, error) {
	filename := version + ".gz"
	f, err := e2eTestListFS.Open(filepath.Join("e2e", "testLists", filename))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open embedded file %v", version)
	}

	logrus.Tracef("Using embedded file %v to obtain E2E test list", filename)
	r, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to processess file %v as a gzip reader", filename)
	}
	defer r.Close()
	return testsFromReader(r)
}

func testsFromReader(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)

	// Scan once to check if reader starts with the prefix.
	scanner.Scan()
	if strings.HasPrefix(scanner.Text(), "#") {
		version := strings.TrimPrefix(scanner.Text(), "#")
		err := &testListStubError{ReferenceVersion: version}

		for scanner.Scan() {
			switch {
			case strings.HasPrefix(scanner.Text(), "+"):
				err.AddTests = append(err.RemoveTests, strings.TrimPrefix(scanner.Text(), "+"))
			case strings.HasPrefix(scanner.Text(), "-"):
				err.RemoveTests = append(err.RemoveTests, strings.TrimPrefix(scanner.Text(), "-"))
			}
		}
		return nil, err
	}

	// Normal file handling, no prefixes to handle.
	tests := []string{scanner.Text()}
	for scanner.Scan() {
		tests = append(tests, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read tests from input")
	}
	return tests, nil
}

func getTestsOnline(baseURL, version string) ([]string, error) {
	c := http.Client{
		Timeout: 10 * time.Second,
	}
	listURL := testURL(baseURL, version)
	logrus.Tracef("Using URL %v to obtain E2E test list", listURL)
	resp, err := c.Get(listURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to GET URL %q, attempt 'offline' input if issues persist", listURL)
	}
	if resp.StatusCode >= 400 {
		return nil, errors.Wrapf(err, "unexpected status (%v %v) when attempting to GET URL %q, attempt 'offline' input if issues persist", resp.Status, resp.StatusCode, listURL)
	}

	r, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process response body as gzip reader")
	}
	defer r.Close()
	return testsFromReader(r)
}

func testURL(baseURL, version string) string {
	return fmt.Sprintf("%v/%v.gz", baseURL, version)
}
