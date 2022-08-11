package driver

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/vmware-tanzu/crash-diagnostics/exec"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Local represents a local script running rather than an in-cluster resource. As a result, it DOES NOT
// satisfy the plugin.Interface interface.
// TODO(jschnake): May want to massage the types/interfaces so that the local plugin does satisfy the interface. However,
// the interface was always made with cluster resources in mind so it is quite different.
type Local struct {
	Core
	scriptName               string
	argsFileName             string
	processContext           context.Context
	processContextCancelFunc context.CancelFunc
}

// Run runs a plugin, declaring all resources it needs, and then
// returns.  It does not block and wait until the plugin has finished.
func (l *Local) Run(filename string) error {
	//l.processContext, cancelFunc = context.WithCancel(context.Background())
	file, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "execution failed for file %v", filename)
	}
	return exec.ExecuteFile(file, map[string]string{})
}

// Cleanup cleans up all resources created by the plugin. In a local context we an ensure the process is stopped.
func (l *Local) Cleanup(kubeClient kubernetes.Interface) {
	return
}

// Monitor continually checks for problems in the resources created by a
// plugin (either because it won't schedule, or the image won't
// download, too many failed executions, etc) and sends the errors as
// Result objects through the provided channel. It should return once the context
// is cancelled.
func (l *Local) Monitor(ctx context.Context, kubeClient kubernetes.Interface, availableNodes []v1.Node, resultsCh chan<- *plugin.Result) {
	return
}

// ExpectedResults is an array of Result objects that a plugin should
// expect to submit.
func (l *Local) ExpectedResults(nodes []v1.Node) []plugin.ExpectedResult {
	return []plugin.ExpectedResult{}
}
