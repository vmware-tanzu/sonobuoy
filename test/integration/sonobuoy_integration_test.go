//go:build integration
// +build integration

package integration

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	yaml "gopkg.in/yaml.v2"
)

const (
	defaultSonobuoyPath = "../../sonobuoy"
	bash                = "/bin/bash"
	defaultTestTimeout  = 2 * time.Minute
)

var (
	// Path to the Sonobuoy CLI
	sonobuoy string

	// A tmp dir to imitate a typical users HOME directory. Useful due to plugin cache logic which
	// typically requires _some_ way to determine a home directory.
	testHome string

	update = flag.Bool("update", false, "update .golden files")
)

func findSonobuoyCLI() (string, error) {
	sonobuoyPath := os.Getenv("SONOBUOY_CLI")
	if sonobuoyPath == "" {
		sonobuoyPath = defaultSonobuoyPath
	}
	if _, err := os.Stat(sonobuoyPath); os.IsNotExist(err) {
		return "", err
	}

	return sonobuoyPath, nil
}

// runSonobuoyCommandWithContext runs the Sonobuoy CLI with the given context and arguments.
// It returns any encountered error and the stdout and stderr from the command execution.
func runSonobuoyCommandWithContext(ctx context.Context, t *testing.T, args string, env ...string) (bytes.Buffer, error) {
	var combinedOutput bytes.Buffer

	command := exec.CommandContext(ctx, sonobuoy, strings.Fields(args)...)
	command.Stdout = &combinedOutput
	command.Stderr = &combinedOutput

	// Test with features on so we can get more feedback. Low risk that
	// this will hide default behavior but in that case we may just make the
	// experimental feature the default then.
	command.Env = []string{"SONOBUOY_ALL_FEATURES=true", "KUBECONFIG=" + os.Getenv("KUBECONFIG"), "HOME=" + testHome}
	for _, v := range env {
		command.Env = append(command.Env, v)
	}
	t.Logf("Running %q with env: %v\n", command.String(), command.Env)

	return combinedOutput, command.Run()
}

func mustRunSonobuoyCommand(t *testing.T, args string) bytes.Buffer {
	return mustRunSonobuoyCommandWithContext(context.Background(), t, args)
}

// mustRunSonobuoyCommandWithContext runs the Sonobuoy CLI with the given context and arguments.
// It returns stdout and fails the test immediately if there are any errors.
func mustRunSonobuoyCommandWithContext(ctx context.Context, t *testing.T, args string, env ...string) bytes.Buffer {
	var stdout, stderr bytes.Buffer

	command := exec.CommandContext(ctx, sonobuoy, strings.Fields(args)...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	// Test with features on so we can get more feedback. Low risk that
	// this will hide default behavior but in that case we may just make the
	// experimental feature the default then.
	command.Env = []string{"SONOBUOY_ALL_FEATURES=true", "KUBECONFIG=" + os.Getenv("KUBECONFIG"), "HOME=" + testHome}
	for _, v := range env {
		command.Env = append(command.Env, v)
	}

	t.Logf("Running %q with env: %v\n", command.String(), command.Env)
	if err := command.Run(); err != nil {
		t.Fatalf("Expected %q to not error but got error: %q with stdout: %q and stderr: %q", args, err, stdout.String(), stderr.String())
	}

	return stdout
}

// runSonobuoyCommand runs the Sonobuoy CLI with the given arguments and a background context.
// It returns any encountered error and the stdout and stderr from the command execution.
func runSonobuoyCommand(t *testing.T, args string) (bytes.Buffer, error) {
	return runSonobuoyCommandWithContext(context.Background(), t, args)
}

// getNamespace returns the namespace to use for the current test and a function to clean it up
// asynchronously afterwards.
func getNamespace(t *testing.T) (string, func()) {
	ns := "sonobuoy-" + strings.ToLower(t.Name())
	return ns, func() { cleanup(t, ns) }
}

// cleanup runs sonobuoy delete for the given namespace. If no namespace is provided, it will
// omit the namespace argument and use the default.
func cleanup(t *testing.T, namespace string) {
	args := "delete"
	if namespace != "" {
		args += " -n " + namespace
	}

	out, err := runSonobuoyCommand(t, args)
	if err != nil {
		t.Logf("Error encountered during cleanup: %q\n", err)
		t.Log(out.String())
	}
}

func TestUseNamespaceFromManifest(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	genArgs := fmt.Sprintf("gen -p testImage/yaml/job-junit-passing-singlefile.yaml -n %v", ns)
	genStdout := mustRunSonobuoyCommandWithContext(ctx, t, genArgs)

	// Write the contents of gen to a temp file
	tmpfile, err := ioutil.TempFile("", "gen.*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(genStdout.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Pass the gen output to sonobuoy run
	runArgs := fmt.Sprintf("run --wait -f %v", tmpfile.Name())
	mustRunSonobuoyCommandWithContext(ctx, t, runArgs)
}

// TestSimpleRun runs a simple plugin to check that it runs successfully
func TestSimpleRun(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-junit-passing-singlefile.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)
}

// TestRetrieveAndExtractWithPodLogs tests that we are able to extract the files
// from the tarball via the retrieve command. It also ensures that we dont
// regress on #1415, that plugin pod logs should be gathered.
func TestRetrieveAndExtractWithPodLogs(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-junit-passing-singlefile.yaml -p testImage/yaml/ds-junit-passing-tar.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	// Create tmpdir and extract contents into it
	tmpdir, err := ioutil.TempDir("", "TestRetrieveAndExtract")
	if err != nil {
		t.Fatal("Failed to create tmp dir")
	}
	defer os.RemoveAll(tmpdir)
	args = fmt.Sprintf("retrieve %v -n %v --extract", tmpdir, ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	// Check that the files are there. Lots of ways to test this but I'm simply going to check that we have
	// a "lot" of files.
	files := []string{}
	err = filepath.Walk(tmpdir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			files = append(files, path)
			return nil
		})
	if err != nil {
		t.Fatalf("Failed to walk path to check: %v", err)
	}

	// Verbose logging here in case we want to just see if certain files were found. Can remove
	// this and just log on error if it is too much.
	t.Logf("Extracted files:\n%v", strings.Join(files, "\n\t-"))
	if len(files) < 20 {
		t.Errorf("Expected many files to be extracted into %v, but only got %v", tmpdir, len(files))
	}

	// This is the logic that ensures that multiple pod logs were gathered.
	podLogCount := 0
	for _, f := range files {
		if strings.HasPrefix(f, filepath.Join(tmpdir, "podlogs", ns)) &&
			strings.HasSuffix(f, "/logs") {
			podLogCount++
		}
	}
	// Should have one for each node/plugin combo expected. Here 2 for the daemonset, the aggregator,
	// and the job.
	if podLogCount < 4 {
		t.Errorf("Expected 4 pod logs to be gathered (2 for the daemonset, aggregator, and job) but only got %v", podLogCount)
	}
}

// TestQuick runs a real "--mode quick" check against the cluster to ensure that it passes.
func TestQuick(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait --mode=quick -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	checkStatusForPluginErrors(ctx, t, ns, "e2e", 0)
	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb)

	checkTarballPluginForErrors(t, tb, "e2e", 0)
}

func TestConfigmaps(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-junit-singlefile-configmap.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)
	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb)

	// Retrieve the sonobuoy results file from the tarball
	resultsArgs := fmt.Sprintf("results %v --plugin %v --mode dump", tb, "job-junit-singlefile-configmap")
	resultsYaml := mustRunSonobuoyCommandWithContext(ctx, t, resultsArgs)
	var resultItem results.Item
	yaml.Unmarshal(resultsYaml.Bytes(), &resultItem)
	expectedStatus := "passed"
	if resultItem.Status != expectedStatus {
		t.Errorf("Expected plugin to have status: %v, got %v", expectedStatus, resultItem.Status)
	}
}

func checkStatusForPluginErrors(ctx context.Context, t *testing.T, ns, plugin string, failCount int) {
	var expectVals []string

	switch {
	case failCount == 0:
		expectVals = []string{
			`"status":"complete","result-status":"passed"`,
			`"passed":1`,
		}
	case failCount > 0:
		expectVals = []string{
			`"status":"complete","result-status":"failed"`,
			fmt.Sprintf(`"failed":%v`, failCount),
		}
	case failCount < 0:
		t.Fatalf("failCount < 0 not permitted; expected >=0, got %v", failCount)
	}

	args := fmt.Sprintf(`status --json -n %v`, ns)
	out := mustRunSonobuoyCommandWithContext(ctx, t, args)
	for _, v := range expectVals {
		if !strings.Contains(out.String(), v) {
			t.Errorf("Expected output of %q to contain %q but output was %v", args, v, out.String())
		}
	}
}

func mustDownloadTarball(ctx context.Context, t *testing.T, ns string) string {
	args := fmt.Sprintf("retrieve -n %v", ns)
	tarballName := mustRunSonobuoyCommandWithContext(ctx, t, args)
	t.Logf("Tarball downloaded to: %v", tarballName.String())
	return strings.TrimSpace(tarballName.String())
}

// checkPluginForErrors runs multiple checks to ensure that failCount errors occurred for the given
// plugin. Ensures that all our different reporting methods are in agreement.
func checkTarballPluginForErrors(t *testing.T, tarball, plugin string, failCount int) {
	expectOut := fmt.Sprintf("Failed: %v", failCount)
	args := fmt.Sprintf("results %v --plugin %v", tarball, plugin)
	out := mustRunSonobuoyCommand(t, args)
	if !strings.Contains(out.String(), expectOut) {
		t.Errorf("Expected output of %q to contain %q but output was %v", args, expectOut, out.String())
	}
}

func saveToArtifacts(t *testing.T, p string) (newPath string) {
	artifactsDir := os.Getenv("ARTIFACTS_DIR")
	if artifactsDir == "" {
		t.Logf("Skipping saving artifact %v since ARTIFACTS_DIR is unset.", p)
		return p
	}

	artifactFile := filepath.Join(artifactsDir, filepath.Base(p))
	origFile := filepath.Join(pwd(t), filepath.Base(p))

	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Logf("Error creating directory %v: %v", artifactsDir, err)
		return p
	}

	var stdout, stderr bytes.Buffer

	// Shell out to `mv` instead of using os.Rename(); the latter caused a problem due to files being on different devices.
	cmd := exec.CommandContext(context.Background(), bash, "-c", fmt.Sprintf("mv %v %v", origFile, artifactFile))
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	t.Logf("Running %q\n", cmd.String())

	if err := cmd.Run(); err != nil {
		t.Logf("Error saving tarball to artifacts directory: %v", err)
		t.Logf("  stdout: %v stderr: %v", stdout.String(), stderr.String())
		return p
	}

	t.Logf("Moved tarball from %q to %q for artifact preservation", origFile, artifactFile)
	return artifactFile
}

// TestSonobuoyVersion checks that all fields in the output from `version` are non-empty
func TestSonobuoyVersion(t *testing.T) {
	t.Parallel()
	stdout := mustRunSonobuoyCommand(t, "version")

	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		versionComponents := strings.Split(line, ":")
		// If a Kubeconfig is not provided, a warning is included that the API version check is skipped.
		// Only check lines where a split on ":" actually happened.
		if len(versionComponents) == 2 && strings.TrimSpace(versionComponents[1]) == "" {
			t.Errorf("expected value for %v to be set, but was empty", versionComponents[0])
		}
	}
}

func TestManualResultsJob(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-manual.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb)

	// Retrieve the sonobuoy results file from the tarball
	resultsArgs := fmt.Sprintf("results %v --plugin %v --mode dump", tb, "job-manual")
	resultsYaml := mustRunSonobuoyCommandWithContext(ctx, t, resultsArgs)
	var resultItem results.Item
	yaml.Unmarshal(resultsYaml.Bytes(), &resultItem)
	expectedStatus := "manual-results-1: 1, manual-results-2: 1"
	if resultItem.Status != expectedStatus {
		t.Errorf("Expected plugin to have status: %v, got %v", expectedStatus, resultItem.Status)
	}
}

func TestManualResultsDaemonSet(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/ds-manual.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb)

	// Retrieve the sonobuoy results file from the tarball
	resultsArgs := fmt.Sprintf("results %v --plugin %v --mode dump", tb, "ds-manual")
	resultsYaml := mustRunSonobuoyCommandWithContext(ctx, t, resultsArgs)
	var resultItem results.Item
	yaml.Unmarshal(resultsYaml.Bytes(), &resultItem)

	// Each status should be reported n times where n is the number of nodes in the cluster.
	// The number of nodes can be determined by the length of the items array in the resultItem as there is an
	// entry for every node where the plugin ran.
	numNodes := len(resultItem.Items)
	expectedStatus := fmt.Sprintf("manual-results-1: %v, manual-results-2: %v", numNodes, numNodes)
	if resultItem.Status != expectedStatus {
		t.Errorf("Expected plugin to have status: %v, got %v", expectedStatus, resultItem.Status)
	}
}

func TestManualResultsWithNestedDetails(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/manual-with-arbitrary-details.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb)

	// Retrieve the sonobuoy results file from the tarball
	resultsArgs := fmt.Sprintf("results %v --plugin %v --mode dump", tb, "manual-with-arbitrary-details")
	resultsYaml := mustRunSonobuoyCommandWithContext(ctx, t, resultsArgs)

	var resultItem results.Item
	yaml.Unmarshal(resultsYaml.Bytes(), &resultItem)

	if len(resultItem.Items) != 1 {
		t.Fatalf("unexpected number of Items in results map, expected 1, got %v", len(resultItem.Items))
	}

	actualDetails := resultItem.Items[0].Details
	expectedDetails := map[string]interface{}{
		"nested-data": map[interface{}]interface{}{
			"nested-key": "value",
		},
	}

	if !reflect.DeepEqual(expectedDetails, actualDetails) {
		t.Errorf("unexpected value for details map, expected %q, got %q", expectedDetails, actualDetails)
	}
}

func TestMain(m *testing.M) {
	var err error
	sonobuoy, err = findSonobuoyCLI()
	if err != nil {
		fmt.Printf("Skipping integration tests: failed to find sonobuoy CLI: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Using Sonobuoy CLI at %q\n", sonobuoy)

	// Creating so we get a clean location for HOME; important due to the plugin cache logic.
	testHome, err = ioutil.TempDir("", "sonobuoy_int_test_home_*")
	if err != nil {
		fmt.Printf("Failed to create tmp dir home: %v", err)
		os.Exit(1)
	}
	defer os.RemoveAll(testHome)

	result := m.Run()
	os.Exit(result)
}

func pwd(t *testing.T) string {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get pwd: %v", err)
	}
	return pwd
}

// TestExactOutput is to test things which can expect exact output; so do not use it
// for things like configs which include timestamps or UUIDs.
func TestExactOutput_LocalGolden(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	testCases := []struct {
		desc       string
		cmdLine    string
		expectFile string
	}{
		{
			desc:       "gen plugin e2e",
			cmdLine:    "gen plugin e2e --kubernetes-version=v123.456.789",
			expectFile: "testdata/gen-plugin-e2e.golden",
		}, {
			desc:       "gen plugin e2e respects configmap",
			cmdLine:    "gen plugin e2e --kubernetes-version=v123.456.789 --configmap=testdata/tiny-configmap.yaml",
			expectFile: "testdata/gen-plugin-e2e-configmap.golden",
		}, {
			desc:       "gen plugin e2e with deprecated flag",
			cmdLine:    "gen plugin e2e --kube-conformance-image-version=v123.456.789",
			expectFile: "testdata/gen-plugin-e2e-kube-flag-still-works.golden",
		}, {
			desc:       "gen with static config",
			cmdLine:    "gen --config=testdata/static-config.json --kubernetes-version=v123.456.789",
			expectFile: "testdata/gen-static.golden",
		}, {
			desc:       "gen specify dynamic plugin",
			cmdLine:    "gen --config=testdata/static-config.json --kubernetes-version=v123.456.789 -pe2e",
			expectFile: "testdata/gen-static-only-e2e.golden",
		}, {
			desc: "gen with variable plugin image",
			cmdLine: "gen --config=testdata/static-config.json --kubernetes-version=v123.456.789 " +
				"-p testdata/hello-world.yaml -p testdata/variable-image.yaml",
			expectFile: "testdata/gen-variable-image.golden",
		}, {
			desc:       "gen doesnt provide UUID",
			cmdLine:    "gen --kubernetes-version=ignore",
			expectFile: "testdata/gen-no-uuid.golden",
		}, {
			desc:       "gen config doesnt provide UUID",
			cmdLine:    "gen config",
			expectFile: "testdata/gen-config-no-uuid.golden",
		}, {
			desc:       "gen with config testing fields that also have flags",
			cmdLine:    "gen --config=testdata/subfieldTest.json --kubernetes-version=ignore",
			expectFile: "testdata/gen-config-no-flags.golden",
		}, {
			desc:       "gen with flags targeting nested config fields",
			cmdLine:    "gen -n=cmdlineNS --image-pull-policy=Always --sonobuoy-image=cmdlineimg --timeout=99 --kubernetes-version=ignore",
			expectFile: "testdata/gen-subfield-flags.golden",
		}, {
			desc:       "gen with config then flags targeting subfields",
			cmdLine:    "gen --config=testdata/subfieldTest.json -n cmdlineNS --image-pull-policy=Always --sonobuoy-image=cmdlineimg --timeout=99 --kubernetes-version=ignore",
			expectFile: "testdata/gen-config-then-flags.golden",
		}, {
			desc:       "gen respects kube-conformance-image for both plugin and config issue 1376",
			cmdLine:    "gen --kube-conformance-image=custom-image --kubernetes-version=v9.8.7",
			expectFile: "testdata/gen-issue-1376.golden",
		}, {
			desc:       "e2e-repo-config should cause KUBE_TEST_REPO_LIST env var to match location used for mount",
			cmdLine:    "gen --e2e-repo-config=./testdata/tiny-configmap.yaml --kubernetes-version=ignore",
			expectFile: "testdata/gen-issue-1375.golden",
		}, {
			desc:       "certified conformance should have no skip value",
			cmdLine:    "gen --mode=certified-conformance --kubernetes-version=ignore",
			expectFile: "testdata/gen-issue-1388.golden",
		}, {
			desc:       "gen rerun-failed should work",
			cmdLine:    "gen --rerun-failed testdata/results-4-e2e-failures.tar.gz --kubernetes-version=ignore",
			expectFile: "testdata/gen-rerunfailed-works.golden",
		}, {
			desc:       "gen rerun-failed should err if missing e2e results",
			cmdLine:    "gen --rerun-failed testdata/results-missing-e2e.tar.gz --kubernetes-version=ignore",
			expectFile: "testdata/gen-rerunfailed-missing.golden",
		}, {
			desc:       "gen rerun-failed should err differently if not tarball",
			cmdLine:    "gen --rerun-failed testdata/tiny-configmap.yaml --kubernetes-version=ignore",
			expectFile: "testdata/gen-rerunfailed-not-tarball.golden",
		}, {
			desc:       "gen rerun-failed should err if no failures",
			cmdLine:    "gen --rerun-failed testdata/results-quick-no-failures.tar.gz --kubernetes-version=ignore",
			expectFile: "testdata/gen-rerunfailed-no-failures.golden",
		}, {
			desc:       "gen plugin should should run plugin name validation",
			cmdLine:    "gen plugin -n badcharS -i foo",
			expectFile: "testdata/gen-plugin-nobadchars.golden",
		}, {
			desc:       "gen should run plugin name validation",
			cmdLine:    "gen -p testdata/plugins/badpluginname.yaml --kubernetes-version=ignore",
			expectFile: "testdata/gen-nobadchars.golden",
		}, {
			desc:       "gen with security context none",
			cmdLine:    "gen --security-context-mode=none --kubernetes-version=ignore",
			expectFile: "testdata/gen-security-context-none.golden",
		}, {
			desc:       "allow plugin renaming",
			cmdLine:    "gen -p testdata/hello-world.yaml@goodbye -p testImage/yaml/job-junit-passing-singlefile.yaml@customname --kubernetes-version=ignore",
			expectFile: "testdata/gen-plugin-renaming.golden",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Allow errors here since we also may test stderr
			output, _ := runSonobuoyCommandWithContext(ctx, t, tc.cmdLine)
			checkFileMatchesOrUpdate(t, output.String(), tc.expectFile, "")
		})
	}
}

func TestOutputIncludes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	testCases := []struct {
		desc          string
		cmdLine       string
		expectSnippet string
	}{
		{
			desc:          "gen with flags targeting subfields then config",
			cmdLine:       "gen -n=cmdlineNS --image-pull-policy=Always --sonobuoy-image=cmdlineimg --timeout=99 --config=testdata/subfieldTest.json",
			expectSnippet: "if a custom config file is set, it must be set before other flags that modify configuration fields",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Allow errors here since we also may test stderr
			output, _ := runSonobuoyCommandWithContext(ctx, t, tc.cmdLine)

			if !strings.Contains(output.String(), tc.expectSnippet) {
				t.Errorf("Expected output to include %q, instead got:\n\n%v", tc.expectSnippet, output.String())
			}
		})
	}
}

// TestPluginCmds will test exact output but will also require other steps to properly
// setup/cleanup so it was split into its own test.
func TestPluginCmds_LocalGolden(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	testCases := []struct {
		desc       string
		cmdLine    string
		expectFile string
		useDir     string
		cleanup    bool
	}{
		{
			desc:       "plugin list",
			cmdLine:    "plugin list",
			useDir:     "testdata/plugins/good",
			expectFile: "testdata/plugin-list.golden",
		}, {
			desc:       "plugin show without ext",
			cmdLine:    "plugin show hello-world",
			useDir:     "testdata/plugins/good",
			expectFile: "testdata/plugin-show-wo-ext.golden",
		}, {
			desc:       "plugin show with ext",
			cmdLine:    "plugin show hello-world.yaml",
			useDir:     "testdata/plugins/good",
			expectFile: "testdata/plugin-show-w-ext.golden",
		}, {
			desc: "plugin show not found",
			// Set --level=panic to avoid timestamp in output.
			cmdLine:    "plugin show no-plugin --level=panic",
			useDir:     "testdata/plugins/good",
			expectFile: "testdata/plugin-show-not-found.golden",
		}, {
			desc:       "plugin show second plugin",
			cmdLine:    "plugin show hw-2",
			useDir:     "testdata/plugins/good",
			expectFile: "testdata/plugin-show-2.golden",
		},
	}

	for _, tc := range testCases {
		origval := os.Getenv("SONOBUOY_DIR")
		defer os.Setenv("SONOBUOY_DIR", origval)

		t.Run(tc.desc, func(t *testing.T) {
			if tc.cleanup {
				defer os.RemoveAll(tc.useDir)
			}

			// Allow errors here since we also may test stderr
			output, _ := runSonobuoyCommandWithContext(ctx, t, tc.cmdLine, "SONOBUOY_DIR="+tc.useDir)
			checkFileMatchesOrUpdate(t, output.String(), tc.expectFile, "")
		})
	}
}

func TestPluginComplexCmds_LocalGolden(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	testCases := []struct {
		desc       string
		cmdLine    []string
		expectFile string

		// Just a workaround so I could still put this test case in this TDT.
		// Will add a non-parsable yaml file in those files listed here before
		// other commands are run.
		addBadPlugins []string
	}{
		{
			desc: "plugin install",
			cmdLine: []string{
				"plugins list",
				"plugin install foo ./testdata/plugins/good/hello-world.yaml",
				"plugins list",
			},
			expectFile: "testdata/plugin-install.golden",
		}, {
			desc: "plugin install then delete",
			cmdLine: []string{
				"plugins list",
				"plugin install foo ./testdata/plugins/good/hello-world.yaml",
				"plugin install foo2 ./testdata/plugins/good/hello-world.yaml",
				"plugins list",
				"plugin uninstall foo",
				"plugins list",
			},
			expectFile: "testdata/plugin-install-delete.golden",
		}, {
			desc: "plugin delete not found",
			cmdLine: []string{
				"plugins uninstall foo",
			},
			expectFile: "testdata/plugin-delete-not-found.golden",
		}, {
			desc: "plugin list doesnt stop on errors",
			cmdLine: []string{
				"plugins list",
				"plugin install p1 ./testdata/plugins/good/hello-world.yaml",
				"plugin install p3 ./testdata/plugins/good/hello-world.yaml",
				"plugin install p5 ./testdata/plugins/good/hello-world.yaml",
				"plugins list",
			},
			addBadPlugins: []string{"p2.yaml", "p4.yaml"},
			expectFile:    "testdata/plugin-list-good-bad.golden",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "sonobuoy_plugin_test_*")
			if err != nil {
				t.Fatal("Failed to create tmp dir")
			}
			defer os.RemoveAll(tmpDir)

			for _, v := range tc.addBadPlugins {
				if err := os.WriteFile(filepath.Join(tmpDir, v), []byte("a:b:c:d:bad:file"), 0777); err != nil {
					t.Fatalf("failed to setup bad plugins for test: %v", err)
				}
			}

			// Allow errors here since we also may test stderr
			var allOutput bytes.Buffer
			for _, cmd := range tc.cmdLine {
				output, _ := runSonobuoyCommandWithContext(ctx, t, cmd, "SONOBUOY_DIR="+tmpDir)
				output.WriteTo(&allOutput)
			}

			checkFileMatchesOrUpdate(t, allOutput.String(), tc.expectFile, tmpDir)
		})
	}
}

// TestPluginLoading_LocalGolden uses gen/goldenfile flow as the easiest/best way
// to test that plugins are loaded from the correct places.
func TestPluginLoading_LocalGolden(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	installedPluginFile := "./testdata/plugin-loading-installed.golden"
	localPluginFile := "./testdata/plugin-loading-local.golden"

	tmpDir, err := ioutil.TempDir("", "sonobuoy_plugin_test_*")
	if err != nil {
		t.Fatalf("Failed to create tmp dir home: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envVars := []string{"SONOBUOY_DIR=" + tmpDir}

	_, e := runSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	if e == nil {
		t.Fatal("Expected a failure since no plugin was installed but got none")
	}

	_ = mustRunSonobuoyCommandWithContext(ctx, t, "plugin install hello-world.yaml ./testdata/plugins/good/hello-world.yaml", envVars...)
	output := mustRunSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	checkFileMatchesOrUpdate(t, output.String(), installedPluginFile, tmpDir)

	_ = mustRunSonobuoyCommandWithContext(ctx, t, "plugin uninstall hello-world.yaml", envVars...)
	_, e = runSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	if e == nil {
		t.Fatal("Expected a failure since no plugin was installed but got none")
	}

	// Copy file to pwd
	input, err := ioutil.ReadFile("./testdata/plugins/good/hello-world.yaml")
	if err != nil {
		t.Fatalf("Failed to read plugin to test pwd loading")
	}

	// Create difference between local/installed plugin so we can differentiate them.
	err = ioutil.WriteFile("hello-world.yaml", bytes.Replace(input, []byte("foo.com"), []byte("localfile.com"), -1), 0644)
	if err != nil {
		t.Fatalf("Failed to copy hello-world to pwd: %v", err)
	}
	defer os.Remove("hello-world.yaml")

	output = mustRunSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	checkFileMatchesOrUpdate(t, output.String(), localPluginFile, tmpDir)

	_ = mustRunSonobuoyCommandWithContext(ctx, t, "plugin install hello-world.yaml ./testdata/plugins/good/hello-world.yaml", envVars...)
	output = mustRunSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	checkFileMatchesOrUpdate(t, output.String(), installedPluginFile, tmpDir)

	// Disable the feature explicitly and ensure we aren't using it.
	envVars = append(envVars, "SONOBUOY_PLUGIN_INSTALLATION=false")
	output = mustRunSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	checkFileMatchesOrUpdate(t, output.String(), localPluginFile, tmpDir)

	if err := os.Remove("hello-world.yaml"); err != nil {
		t.Fatalf("Failed to delete local file: %v", err)
	}

	_, e = runSonobuoyCommandWithContext(ctx, t, "gen -p hello-world.yaml --kubernetes-version=v123.456.789", envVars...)
	if e == nil {
		t.Fatal("Expected a failure since no plugin was installed but got none")
	}
}

func checkFileMatchesOrUpdate(t *testing.T, output, expectFile, maskDir string) {
	binaryVersion := mustRunSonobuoyCommand(t, "version --short")
	binaryVer := strings.TrimSpace(binaryVersion.String())

	outString := strings.ReplaceAll(output, binaryVer, "*STATIC_FOR_TESTING*")
	outString = strings.ReplaceAll(outString, testHome, "*STATIC_FOR_TESTING*")
	if maskDir != "" {
		outString = strings.ReplaceAll(outString, maskDir, "*STATIC_FOR_TESTING*")
	}
	r := regexp.MustCompile("time=\".*?\"")
	outString = r.ReplaceAllString(outString, `time="STATIC_TIME_FOR_TESTING"`)

	if *update {
		if err := os.WriteFile(expectFile, []byte(outString), 0666); err != nil {
			t.Fatalf("Failed to update goldenfile: %v", err)
		}
	} else {
		fileData, err := ioutil.ReadFile(expectFile)
		if err != nil {
			t.Fatalf("Failed to read golden file %v: %v", expectFile, err)
		}
		if diff := pretty.Compare(string(fileData), outString); diff != "" {
			t.Errorf("Expected output to equal goldenfile: %v but got diff:\n\n%v", expectFile, diff)
		}
	}
}
