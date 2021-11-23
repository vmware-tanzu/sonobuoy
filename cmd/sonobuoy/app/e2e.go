package app

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"
)

type e2eFlags struct {
	e2efocus          string
	e2eskip           []string
	k8sVersion        image.ConformanceImageVersion
}

func NewCmdE2E() *cobra.Command {
	f := e2eFlags{}
	cmd := &cobra.Command{
		Use:   "e2e",
		Short: "Generates a list of all tests and tags in that tests",
		Run:   e2eSonobuoyRun(&f),
		Args:  cobra.ExactArgs(0),
	}

	return cmd
}
