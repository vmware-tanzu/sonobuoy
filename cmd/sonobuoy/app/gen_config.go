package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

// NewCmdGenConfig creates the `config` command which will print out
// the default sonobuoy config in a json format.
func NewCmdGenConfig() *cobra.Command {
	var f genFlags

	var GenCommand = &cobra.Command{
		Use:   "config",
		Short: "Generates a sonobuoy config for input to sonobuoy gen or run.",
		Run:   genConfigCobra(&f),
		Args:  cobra.ExactArgs(0),
	}
	GenCommand.Flags().AddFlagSet(GenFlagSet(&f, EnabledRBACMode))

	return GenCommand
}

// genConfigCobra is the function that executes the functional logic
// but wraps it in the function signature/flow expected for cobra.
func genConfigCobra(f *genFlags) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		s, err := genConfig(f)
		if err != nil {
			errlog.LogError(err)
			os.Exit(1)
		}
		fmt.Println(string(s))
	}
}

// genConfig is the actual functional logic of the command.
func genConfig(f *genFlags) ([]byte, error) {
	// Using genflags instead of config.New() because
	// it will include any defaults we have on the command line such
	// as default plugin selection. We didn't want to wire this into
	// the `config` package, but it will be a default value the CLI
	// users expect.
	b, err := json.Marshal(f.sonobuoyConfig.Get())
	return b, errors.Wrap(err, "unable to marshal configuration")
}
