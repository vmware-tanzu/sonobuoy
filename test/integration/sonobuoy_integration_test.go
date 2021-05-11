// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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
func runSonobuoyCommandWithContext(ctx context.Context, t *testing.T, args string) (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer

	command := exec.CommandContext(ctx, sonobuoy, strings.Fields(args)...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	t.Logf("Running %q\n", command.String())

	return stdout, stderr, command.Run()
}

func mustRunSonobuoyCommand(t *testing.T, args string) bytes.Buffer {
	return mustRunSonobuoyCommandWithContext(context.Background(), t, args)
}

// mustRunSonobuoyCommandWithContext runs the Sonobuoy CLI with the given context and arguments.
// It returns stdout and fails the test immediately if there are any errors.
func mustRunSonobuoyCommandWithContext(ctx context.Context, t *testing.T, args string) bytes.Buffer {
	var stdout, stderr bytes.Buffer

	command := exec.CommandContext(ctx, sonobuoy, strings.Fields(args)...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	t.Logf("Running %q\n", command.String())
	if err := command.Run(); err != nil {
		t.Fatalf("Expected %q to not error but got error: %q with stdout: %q and stderr: %q", args, err, stdout.String(), stderr.String())
	}

	return stdout
}

// runSonobuoyCommand runs the Sonobuoy CLI with the given arguments and a background context.
// It returns any encountered error and the stdout and stderr from the command execution.
func runSonobuoyCommand(t *testing.T, args string) (bytes.Buffer, bytes.Buffer, error) {
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

	stdout, stderr, err := runSonobuoyCommand(t, args)
	if err != nil {
		t.Logf("Error encountered during cleanup: %q\n", err)
		t.Log(stdout.String())
		t.Log(stderr.String())
	}
}

func TestUseNamespaceFromManifest(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-junit-passing-singlefile.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)
}

// TestQuick runs a real "--mode quick" check against the cluster to ensure that it passes.
func TestQuick(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait --mode=quick -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	checkStatusForPluginErrors(ctx, t, ns, "e2e", 0)
	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb, t.Name()+".tar.gz")

	checkTarballPluginForErrors(t, tb, "e2e", 0)
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
	if plugin == "e2e" {
		expectOut := fmt.Sprintf("failed tests: %v", failCount)
		args := fmt.Sprintf("e2e %v ", tarball)
		out := mustRunSonobuoyCommand(t, args)
		if !strings.Contains(out.String(), expectOut) {
			t.Errorf("Expected output of %q to contain %q but output was %v", args, expectOut, out.String())
		}
	}

	expectOut := fmt.Sprintf("Failed: %v", failCount)
	args := fmt.Sprintf("results %v --plugin %v", tarball, plugin)
	out := mustRunSonobuoyCommand(t, args)
	if !strings.Contains(out.String(), expectOut) {
		t.Errorf("Expected output of %q to contain %q but output was %v", args, expectOut, out.String())
	}
}

func saveToArtifacts(t *testing.T, targetFile, newFilename string) (newPath string) {
	artifactsDir := os.Getenv("ARTIFACTS_DIR")
	if artifactsDir == "" {
		t.Logf("Skipping saving artifact %v since ARTIFACTS_DIR is unset.", targetFile)
		return targetFile
	}

	// Default to the same filename.
	if newFilename == "" {
		newFilename = targetFile
	}

	artifactFile := filepath.Join(artifactsDir, filepath.Base(newFilename))
	origFile := filepath.Join(pwd(t), filepath.Base(targetFile))

	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Logf("Error creating directory %v: %v", artifactsDir, err)
		return targetFile
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
		return targetFile
	}

	t.Logf("Moved tarball from %q to %q for artifact preservation", origFile, artifactFile)
	return artifactFile
}

// TestSonobuoyVersion checks that all fields in the output from `version` are non-empty
func TestSonobuoyVersion(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-manual.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb, t.Name()+".tar.gz")

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
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/ds-manual.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb, t.Name()+".tar.gz")

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
	ctx, cancel := context.WithTimeout(context.Background(), defaultTestTimeout)
	defer cancel()

	ns, cleanup := getNamespace(t)
	defer cleanup()

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/manual-with-arbitrary-details.yaml -n %v", ns)
	mustRunSonobuoyCommandWithContext(ctx, t, args)

	tb := mustDownloadTarball(ctx, t, ns)
	tb = saveToArtifacts(t, tb, t.Name()+".tar.gz")

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
