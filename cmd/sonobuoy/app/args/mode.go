package args

import (
	"fmt"
	"strings"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/spf13/cobra"
)

const (
	// Quick runs a single E2E test and the systemd log tests
	Quick Mode = "quick"
	// Conformance runs all of the E2E tests and the systemd log tests
	Conformance Mode = "conformance"
	// Extended run all of the E2E tests, the systemd log tests, and
	// Heptio's E2E Tests
	Extended Mode = "extended"
)

// Mode identifies a specific mode
type Mode string

var modeMap = map[string]Mode{
	"conformance": Conformance,
	"quick":       Quick,
	"extended":    Extended,
}

// ModeConfig represents the sonobuoy configuration for a given mode
type ModeConfig struct {
	// E2EFocus is the string to be passed to the E2EFOCUS env var
	E2EFocus string
	// Selectors are the plugins selected by thi mode
	Selectors []plugin.Selection
}

// AddModeFlag adds a mode flag to existing command
func AddModeFlag(modeMode *Mode, cmd *cobra.Command) {
	cmd.PersistentFlags().VarP(
		modeMode, "mode", "m",
		fmt.Sprintf(
			"What mode to run sonobuoy in. One of %s (default Conformance).",
			strings.Join(getModes(), ", "),
		),
	)
}

// String needed for pflag.Value
func (n *Mode) String() string { return string(*n) }

// Type needed for pflag.Value
func (n *Mode) Type() string { return "Mode" }

// Set the name with a given string. Returns error on unknown mode
func (n *Mode) Set(str string) error {
	mode, ok := modeMap[str]
	if !ok {
		return fmt.Errorf("unknown mode %s", str)
	}
	*n = mode
	return nil
}

// Get returns the ModeConfig associated with a mode name, or nil
// if there's no associated mode
func (n *Mode) Get() *ModeConfig {
	// Default value
	name := Conformance
	if n != nil && *n != "" {
		name = *n
	}
	switch name {
	case Conformance:
		return &ModeConfig{
			E2EFocus: "Conformance",
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case Quick:
		return &ModeConfig{
			E2EFocus: "Pods should be submitted and removed",
			Selectors: []plugin.Selection{
				{Name: "e2e"},
				{Name: "systemd-logs"},
			},
		}
	case Extended:
		return &ModeConfig{
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

func getModes() []string {
	keys := make([]string, len(modeMap))
	i := 0
	for k := range modeMap {
		keys[i] = k
		i++
	}
	return keys
}
