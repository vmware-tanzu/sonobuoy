// +build integration

package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
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

// TestSonobuoyVersion checks that all fields in the output from `version` are non-empty
func TestSonobuoyVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	command := exec.Command(sonobuoy, "version")
	command.Stdout = &stdout
	command.Stderr = &stderr

	t.Logf("Running %q\n", command.String())
	if err := command.Run(); err != nil {
		t.Errorf("Sonobuoy exited with an error: %v", err)
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
