package utils

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	SonobuoyPod = "sonobuoy"
)

// OutOfClusterClient returns a kubernetes client that is accessing the
// cluster from outside the cluster.
func OutOfClusterClient() (kubernetes.Interface, error) {
	kubeconfig := locateKubeconfig()
	if len(kubeconfig) == 0 {
		return nil, errors.New("Could not locate kubeconfig")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "could not build config from kubeconfig")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not make a new clientset from config")
	}
	return clientset, nil
}

func locateKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		u, err := user.Current()
		if err != nil {
			return ""
		}
		kubeconfig = filepath.Join(u.HomeDir, ".kube", "config")
		// make sure this file exists
		_, err = os.Stat(kubeconfig)
		if err != nil {
			return ""
		}
	}
	return kubeconfig
}
