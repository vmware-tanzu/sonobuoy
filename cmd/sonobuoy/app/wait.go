package app

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type WaitOutputMode string

const (
	SilentOutputMode   WaitOutputMode = "Silent"
	SpinnerOutputMode  WaitOutputMode = "Spinner"
	ProgressOutputMode WaitOutputMode = "Progress"
)

var waitOutputModeMap = map[string]WaitOutputMode{
	string(SilentOutputMode):   SilentOutputMode,
	string(SpinnerOutputMode):  SpinnerOutputMode,
	string(ProgressOutputMode): ProgressOutputMode,
}

// String needed for pflag.Value.
func (w *WaitOutputMode) String() string { return string(*w) }

// Type needed for pflag.Value.
func (w *WaitOutputMode) Type() string { return "string" }

// Set the WaitOutputMode to the given string, or error if it's not a known WaitOutputMode mode.
func (w *WaitOutputMode) Set(str string) error {
	// Allow lowercase on the command line
	upcase := cases.Title(language.AmericanEnglish).String(str)
	mode, ok := waitOutputModeMap[upcase]
	if !ok {
		return fmt.Errorf("unknown wait output mode %s", str)
	}
	*w = mode
	return nil
}
