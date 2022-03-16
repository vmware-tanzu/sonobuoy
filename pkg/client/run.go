/*
Copyright 2018 Heptio Inc.

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

package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
	"golang.org/x/term"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/api/core/v1"
)

const (
	bufferSize          = 4096
	pollInterval        = 20 * time.Second
	spinnerType     int = 14
	spinnerDuration     = 2000 * time.Millisecond
	spinnerColor        = "red"

	// Special key for when to load manifest from stdin instead of a local file.
	stdinFile = "-"
)

var (
	whitespaceRemover = strings.NewReplacer(" ", "", "\t", "")
)

// RunManifest is the same as Run(*RunConfig) execpt that the []byte given
// should represent the output from `sonobuoy gen`, a series of YAML resources
// separated by `---`. This method will disregard the RunConfig.GenConfig
// and instead use the given []byte as the manifest.
func (c *SonobuoyClient) RunManifest(cfg *RunConfig, manifest []byte) error {
	buf := bytes.NewBuffer(manifest)
	d := yaml.NewYAMLOrJSONDecoder(buf, bufferSize)

	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return errors.Wrap(err, "couldn't decode template")
		}

		// Skip over empty or partial objects
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		obj := &unstructured.Unstructured{}
		if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), ext.Raw, obj); err != nil {
			return errors.Wrap(err, "couldn't decode template")
		}
		name, err := c.dynamicClient.Name(obj)
		if err != nil {
			return errors.Wrap(err, "could not get object name")
		}
		namespace, err := c.dynamicClient.Namespace(obj)
		if err != nil {
			return errors.Wrap(err, "could not get object namespace")
		}

		// err is used to determine output for user; but first extract resource
		_, err = c.dynamicClient.CreateObject(obj)
		resource, err2 := c.dynamicClient.ResourceVersion(obj)
		if err2 != nil {
			return errors.Wrap(err, "could not get resource of object")
		}
		if err := handleCreateError(name, namespace, resource, err); err != nil {
			return errors.Wrap(err, "failed to create object")
		}
	}

	if cfg.Wait > time.Duration(0) {
		return c.WaitForRun(cfg)
	}

	return nil
}

// WaitForRun handles the 'wait' from sonobuoy run --wait. "Attaches" to the sonobuoy run
// in the configured namespace and then returns when completed. Returns errors encountered
func (c *SonobuoyClient) WaitForRun(cfg *RunConfig) error {
	// The runCondition will be a closure around this variable so that subsequent
	// polling attempts know if the status has been present yet.
	seenStatus := false

	ns := cfg.GetNamespace()

	// Printer allows us to more easily add output conditionally throughout the runCondition.
	var printer = func(string) {}
	if cfg.WaitOutput == progressMode {
		lastMsg := ""
		seenLines := map[string]struct{}{}
		printer = func(s string) {
			if s == lastMsg {
				fmt.Println("...")
				return
			}
			lines := strings.Split(s, "\n")
			for _, l := range lines {
				lKey := stripSpaces(l)
				if _, ok := seenLines[lKey]; !ok {
					seenLines[lKey] = struct{}{}
					fmt.Printf("%v %v\n", time.Now().Format("15:04:05"), l)
				}
			}
			lastMsg = s
		}
	}
	runCondition := func() (bool, error) {

		// Get the Aggregator pod and check if its status is completed or terminated.
		status, pod, err := c.GetStatusPod(&StatusConfig{Namespace: ns})
		switch {
		case err != nil && seenStatus:
			printer(fmt.Sprintf("Failed to check status of the aggregator: %v", err))
			return false, errors.Wrap(err, "failed to get status")
		case err != nil && pod == nil && !seenStatus:
			// Allow more time for the status to reported.
			printer("Waiting for the aggregator to get tagged with its current status...")
			return false, nil
		case err != nil && pod != nil && !seenStatus:
			// Allow more time for the status to reported, but also report the status of the aggregator pod
			printer(fmt.Sprintf("Waiting for the aggregator status to become %s. Currently the status is %s", corev1.PodRunning, getPodStatus(*pod)))
			return false, nil
		case status != nil:
			seenStatus = true
		}

		// if nil below was added for coverage on staticcheck
		// TODO: ensure this is the desired behavior
		if status == nil {
			printer("Aggregator status is nil after having been applied previously. Report this as an error.")
			return false, nil
		}

		switch {
		case status.Status == aggregation.CompleteStatus:
			printer(aggregatorStatusToProgress(status))
			return true, nil
		case status.Status == aggregation.FailedStatus:
			printer(aggregatorStatusToProgress(status))
			return true, fmt.Errorf("pod entered a fatal terminal status: %v", status.Status)
		}
		printer(aggregatorStatusToProgress(status))
		return false, nil
	}

	switch cfg.WaitOutput {
	case spinnerMode:
		var s = getSpinnerInstance()
		s.Start()
		defer s.Stop()
	case progressMode:
		// handled by conditionFunc since it has to be part of the polling.
		// Could use channels but that will lead to future issues.
	}
	err := wait.Poll(pollInterval, cfg.Wait, runCondition)
	if err != nil {
		return errors.Wrap(err, "waiting for run to finish")
	}
	return nil
}

func stripSpaces(s string) string {
	return whitespaceRemover.Replace(s)
}

// Run will use the given RunConfig to generate YAML for a series of resources and then
// create them in the cluster.
func (c *SonobuoyClient) Run(cfg *RunConfig) error {
	if cfg == nil {
		return errors.New("nil RunConfig provided")
	}

	var manifest []byte
	var err error
	if len(cfg.GenFile) != 0 {
		manifest, err = loadManifestFromFile(cfg.GenFile)
		if err != nil {
			return errors.Wrap(err, "loading manifest")
		}
	} else {
		manifest, err = c.GenerateManifest(&cfg.GenConfig)
		if err != nil {
			return errors.Wrap(err, "couldn't run invalid manifest")
		}
	}

	return c.RunManifest(cfg, manifest)
}

func loadManifestFromFile(f string) ([]byte, error) {
	if f == stdinFile {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("nothing on stdin to read")
		}

		return ioutil.ReadAll(os.Stdin)
	} else {
		return ioutil.ReadFile(f)
	}
}

func handleCreateError(name, namespace, resource string, err error) error {
	log := logrus.WithFields(logrus.Fields{
		"name":      name,
		"namespace": namespace,
		"resource":  resource,
	})

	switch {
	case err == nil:
		log.Info("create request issued")
	// Some resources (like ClusterRoleBinding and ClusterBinding) aren't
	// namespaced and may overlap between runs. So don't abort on duplicate errors
	// in this case.
	case namespace == "" && kubeerror.IsAlreadyExists(err):
		log.Info("object already exists")
	case err != nil:
		return errors.Wrapf(err, "failed to create API resource %s", name)
	}
	return nil
}

func getSpinnerInstance() *spinner.Spinner {
	s := spinner.New(spinner.CharSets[spinnerType], spinnerDuration)
	s.Color(spinnerColor)
	return s
}

func aggregatorStatusToProgress(s *aggregation.Status) string {
	var b bytes.Buffer
	if err := printAll(&b, s); err != nil {
		logrus.Error(errors.Wrap(err, "failed printing aggregator status"))
	}
	return b.String()
}

// TODO(jschnake): printall, defaultTabWriter, and humanReadableStatus are copies from app package.
// Printing is normally the app packages responsibility but the line blurs here.
// Copying code to facilitate progress feature, but should consider extracting from
// both locations to a generic one for reuse.
func printAll(w io.Writer, status *aggregation.Status) error {
	tw := defaultTabWriter(w)

	fmt.Fprintf(tw, "PLUGIN\tNODE\tSTATUS\tRESULT\tPROGRESS\t\n")
	for _, pluginStatus := range status.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t\n", pluginStatus.Plugin, pluginStatus.Node, pluginStatus.Status, pluginStatus.ResultStatus, pluginStatus.Progress.FormatPluginProgress())
	}

	if err := tw.Flush(); err != nil {
		return errors.Wrap(err, "couldn't write status out")
	}

	fmt.Fprintf(w, "\n%s\n", humanReadableStatus(status.Status))
	return nil
}

func defaultTabWriter(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 2, 3, ' ', tabwriter.AlignRight)
}

func humanReadableStatus(str string) string {
	switch str {
	case aggregation.RunningStatus:
		return "Sonobuoy is still running. Runs can take 60 minutes or more depending on cluster and plugin configuration."
	case aggregation.FailedStatus:
		return "Sonobuoy has failed. You can see what happened with `sonobuoy logs`."
	case aggregation.CompleteStatus:
		return "Sonobuoy has completed. Use `sonobuoy retrieve` to get results."
	case aggregation.PostProcessingStatus:
		return "Sonobuoy plugins have completed. Preparing results for download."
	default:
		return fmt.Sprintf("Sonobuoy is in unknown state %q. Please report a bug at github.com/vmware-tanzu/sonobuoy", str)
	}
}


func getPodStatus(pod corev1.Pod) string {
	const ContainersNotReady = "ContainersNotReady"
	if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
		//Scan all the pod.Status.Conditions
		//scan pod.Conditions, and find the first where condition.Status != corev1.ConditionTrue
		for _, condition := range pod.Status.Conditions {
			if condition.Status != corev1.ConditionTrue {
				retval := fmt.Sprintf("Status: %s, Reason: %s, %s", pod.Status.Phase, condition.Reason, condition.Message)
				//If the reason is ContainersNotReady, we can also print information about why the containers are not ready
				if string(condition.Reason) == ContainersNotReady {
					retval += "\nDetails of containers that are not ready:\n"
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if !containerStatus.Ready {
							retval += fmt.Sprintf("%s: ", containerStatus.Name)
							var reason string
							var message string
							if containerStatus.State.Waiting != nil {
								reason = containerStatus.State.Waiting.Reason
								message = containerStatus.State.Waiting.Message
								retval += "waiting: "
							}
							if containerStatus.State.Terminated != nil {
								reason = containerStatus.State.Terminated.Reason
								message = containerStatus.State.Terminated.Message
								retval += "terminated: "
							}
							retval += reason
							//Add state.MEssage only if it isn't blank
							if len(strings.TrimSpace(message)) > 0 {
								retval += ", " + message
							}
							retval += "\n"
						}
					}

				}
				return retval
			}
		}
	}
	//If the status is running or succeeded, we just print the status, although this function might never be called in this case
	return fmt.Sprintf("Status: %s", pod.Status.Phase)
}
