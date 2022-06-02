package app

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

func NewCmdGenCLIDocs() *cobra.Command {
	var docsCmd = &cobra.Command{
		Use:   "cli <output dir>",
		Short: "Generates markdown docs for the CLI",
		Run: func(cmd *cobra.Command, args []string) {
			root := NewSonobuoyCommand()
			err := doc.GenMarkdownTree(root, args[0])
			if err != nil {
				errlog.LogError(err)
			}
		},
		Args:   cobra.ExactArgs(1),
		Hidden: true,
	}
	return docsCmd
}
