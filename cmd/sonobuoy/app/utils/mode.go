package mode

import (
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/spf13/cobra"
)

// Name is the name of the mode

const (
	// Quick runs a single E2E test and the systemd log tests
	Quick Name = "quick"
	// Conformance runs all of the E2E tests and the systemd log tests
	Conformance Name = "conformance"
	// Extended run all of the E2E tests, the systemd log tests, and
	// Heptio's E2E Tests
	Extended Name = "extended"
)

// Name identifies a specific mode
type Name string

// Config is the
type Config struct {
	// E2EFocus is the string to be passed to the E2EFOCUS env var
	E2EFocus string
	// Selectors are the plugins selected by thi mode
	Selectors []plugin.Selection
}

// AddFlag adds a mode flag to existing command
func AddFlag(modeName *Name, cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(
		modeName, "mode", "m",
		"What mode to run sonobuoy in",
	)
}

// String needed for pflag.Value
func (n *Name) String() string { return string(*n) }

// Type needed for pflag.Value
func (n *Name) Type() string { return "Name" }

// Set the name with a given string. Returns error on unknown mode
func (n *Name) Set(str string) error {
	switch str {
	case "conformance":
		*n = Conformance
	case "quick":
		*n = Quick
	case "extended":
		*n = Extended
	case "":
		*n = Conformance
	default:
		return fmt.Errorf("unknown mode %s", str)
	}
	return nil
}

// Get returns the Config associated with a mode name, or nil
// if there's no associated mode
func (n *Name) Get() *Config {
	// Default value
	name := Conformance
	if n != nil && *n != "" {
		name = *n
	}
	switch name {
	case Conformance:
		return &Config{
			E2EFocus: "Conformance",
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case Quick:
		return &Config{
			E2EFocus: "Pods should be submitted and removed",
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case Extended:
		return &Config{
			E2EFocus: "Conformance",
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
				{Name: "heptio-e2e"},
			},
		}
	default:
		return nil
	}
}
