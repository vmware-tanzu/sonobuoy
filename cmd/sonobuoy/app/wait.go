package app

import (
	"fmt"
	"strings"
)

type WaitOutputMode string

const (
	SilentOutputMode  WaitOutputMode = "Silent"
	SpinnerOutputMode WaitOutputMode = "Spinner"
)

var waitOutputModeMap = map[string]WaitOutputMode{
	string(SilentOutputMode):  SilentOutputMode,
	string(SpinnerOutputMode): SpinnerOutputMode,
}

// String needed for pflag.Value.
func (w *WaitOutputMode) String() string { return string(*w) }

// Type needed for pflag.Value.
func (w *WaitOutputMode) Type() string { return "WaitOutputMode" }

// Set the WaitOutputMode to the given string, or error if it's not a known WaitOutputMode mode.
func (w *WaitOutputMode) Set(str string) error {
	// Allow lowercase on the command line
	upcase := strings.Title(str)
	mode, ok := waitOutputModeMap[upcase]
	if !ok {
		return fmt.Errorf("unknown Wait Output mode %s", str)
	}
	*w = mode
	return nil
}
