package operations

import (
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin"
)

// Mode identifies a specific mode
type Mode string

const (
	// Quick runs a single E2E test and the systemd log tests
	Quick Mode = "quick"
	// Conformance runs all of the E2E tests and the systemd log tests
	Conformance Mode = "conformance"
	// Extended run all of the E2E tests, the systemd log tests, and
	// Heptio's E2E Tests
	Extended Mode = "extended"
)

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
	switch *n {
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

func GetModes() []string {
	keys := make([]string, len(modeMap))
	i := 0
	for k := range modeMap {
		keys[i] = k
		i++
	}
	return keys
}
