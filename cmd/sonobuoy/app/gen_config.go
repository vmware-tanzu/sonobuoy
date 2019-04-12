package app

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewCmdGenConfig creates the `config` command which will print out
// the default sonobuoy config in a json format.
func NewCmdGenConfig() *cobra.Command {
	var GenCommand = &cobra.Command{
		Use:   "config",
		Short: "Generates a sonobuoy config for input to sonobuoy gen or run.",
		Run:   genConfigCobra,
		Args:  cobra.ExactArgs(0),
	}

	return GenCommand
}

// genConfigCobra is the function that executes the functional logic
// but wraps it in the function signature/flow expected for cobra.
func genConfigCobra(cmd *cobra.Command, args []string) {
	s, err := genConfig()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
	fmt.Println(string(s))
}

// genConfig is the actual functional logic of the command.
func genConfig() ([]byte, error) {
	// Using genflags.getConfig() instead of config.New() because
	// it will include any defaults we have on the command line such
	// as default plugin selection. We didn't want to wire this into
	// the `config` package, but it will be a default value the CLI
	// users expect.
	c := genflags.resolveConfig()
	b, err := json.Marshal(c)
	return b, errors.Wrap(err, "unable to marshal configuration")
}
