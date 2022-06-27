package app

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/sonobuoy/pkg/errlog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type getFlags struct {
	namespace string
	plugin    string
	kubecfg   Kubeconfig
}

func NewCmdGet() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Fetches Sonobuoy resources of a specified type",
	}

	// Add pod subcommand
	flags := getFlags{}

	podsCmd := &cobra.Command{
		Use:     "pod",
		Short:   "Fetch sonobuoy pods",
		Aliases: []string{"pods"},
		Run: func(cmd *cobra.Command, args []string) {
			if err := getPods(&flags); err != nil {
				errlog.LogError(err)
				os.Exit(1)
			}
		},
	}

	AddNamespaceFlag(&flags.namespace, podsCmd.Flags())
	podsCmd.Flags().StringVarP(&flags.plugin, "plugin", "p", "", "Plugin to locate pods for")

	cmd.AddCommand(podsCmd)

	return cmd
}

func getPods(flags *getFlags) error {
	selector := fmt.Sprintf("component=%s,sonobuoy-component=plugin", flags.namespace)

	if len(flags.plugin) > 0 {
		selector += fmt.Sprintf(",sonobuoy-plugin=%s", flags.plugin)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	sbc, err := getSonobuoyClientFromKubecfg(flags.kubecfg)

	if err != nil {
		return errors.Wrap(err, "could not create sonobuoy client")
	}

	client, err := sbc.Client()

	if err != nil {
		return errors.Wrap(err, "could not retrieve kubernetes client")
	}

	pods, err := client.CoreV1().Pods(flags.namespace).List(context.TODO(), listOptions)

	if err != nil {
		return errors.Wrap(err, "could not retrieve list of pods")
	}

	for _, pod := range pods.Items {
		fmt.Printf("%s\n", pod.GetName())
	}

	return nil
}
