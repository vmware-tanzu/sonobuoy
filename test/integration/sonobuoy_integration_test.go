// +build integration

package integration

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	defaultSonobuoyPath = "../../sonobuoy"
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
func runSonobuoyCommandWithContext(ctx context.Context, t *testing.T, args string) (error, bytes.Buffer, bytes.Buffer) {
	var stdout, stderr bytes.Buffer

	command := exec.CommandContext(ctx, sonobuoy, strings.Fields(args)...)
	command.Stdout = &stdout
	command.Stderr = &stderr

	t.Logf("Running %q\n", command.String())

	return command.Run(), stdout, stderr
}

// runSonobuoyCommand runs the Sonobuoy CLI with the given arguments and a background context.
// It returns any encountered error and the stdout and stderr from the command execution.
func runSonobuoyCommand(t *testing.T, args string) (error, bytes.Buffer, bytes.Buffer) {
	return runSonobuoyCommandWithContext(context.Background(), t, args)
}

// cleanup runs sonobuoy delete for the given namespace. If no namespace is provided, it will
// omit the namespace argument and use the default.
func cleanup(t *testing.T, namespace string) {
	args := "delete"
	if namespace != "" {
		args += " -n " + namespace
	}

	err, stdout, stderr := runSonobuoyCommand(t, args)

	if err != nil {
		t.Logf("Error encountered during cleanup: %q\n", err)
		t.Log(stdout.String())
		t.Log(stderr.String())
	}
}

func TestUseNamespaceFromManifest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ns := "sonobuoy-" + strings.ToLower(t.Name())
	defer cleanup(t, ns)

	genArgs := fmt.Sprintf("gen -p testImage/yaml/job-junit-passing-singlefile.yaml -n %v", ns)
	err, genStdout, genStderr := runSonobuoyCommandWithContext(ctx, t, genArgs)

	if err != nil {
		t.Fatalf("Sonobuoy exited with an error: %q\n", err)
		t.Log(genStderr.String())
	}

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
	err, _, runStderr := runSonobuoyCommandWithContext(ctx, t, runArgs)
	if err != nil {
		t.Errorf("Sonobuoy exited with an error: %q\n", err)
		t.Log(runStderr.String())
	}
}

// TestSimpleRun runs a simple plugin to check that it runs successfully
func TestSimpleRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ns := "sonobuoy-" + strings.ToLower(t.Name())
	defer cleanup(t, ns)

	args := fmt.Sprintf("run --image-pull-policy IfNotPresent --wait -p testImage/yaml/job-junit-passing-singlefile.yaml -n %v", ns)
	err, _, stderr := runSonobuoyCommandWithContext(ctx, t, args)

	if err != nil {
		t.Errorf("Sonobuoy exited with an error: %q\n", err)
		t.Log(stderr.String())
	}
}

// TestSonobuoyVersion checks that all fields in the output from `version` are non-empty
func TestSonobuoyVersion(t *testing.T) {
	err, stdout, stderr := runSonobuoyCommand(t, "version")

	if err != nil {
		t.Errorf("Sonobuoy exited with an error: %q\n", err)
		t.Log(stderr.String())
		t.FailNow()
	}

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

func TestMain(m *testing.M) {
	var err error
	sonobuoy, err = findSonobuoyCLI()
	if err != nil {
		fmt.Printf("Skipping integration tests: failed to find sonobuoy CLI: %v\n", err)
		os.Exit(1)
	}

	result := m.Run()
	os.Exit(result)
}
