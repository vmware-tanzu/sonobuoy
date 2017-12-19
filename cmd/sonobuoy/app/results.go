/*
Copyright 2017 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(resultsCmd)
}

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Copy results from the finished running sonobuoy pod to local disk",
	Run:   runResults,
}

func runResults(cmd *cobra.Command, args []string) {
	// TODO(chuckha) Consider making this customizable.
	namespace := "heptio-sonobuoy"

	// TODO(chuckha) Move away from shelling out if worthwhile
	kubectl := exec.Command("kubectl", "cp", namespace+"/sonobuoy:/tmp/sonobuoy", "./results", "--namespace="+namespace)
	// TODO(chuckha) there may be other ways to get kubeconfig, such as --kubeconfig
	kubectl.Env = []string{os.Getenv("KUBECONFIG")}
	logrus.WithField("KUBECONFIG", os.Getenv("KUBECONFIG")).Info("KUBECONFIG environment")
	logrus.WithField("command", kubectl.Args).Info("command being run")
	logrus.Info("running command")
	output, err := kubectl.CombinedOutput()
	logrus.WithError(err).Info("command finished")
	logrus.WithField("output", string(output)).Info("finished running")
}
