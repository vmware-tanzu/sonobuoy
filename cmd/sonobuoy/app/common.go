package app

import (
	"time"
	"github.com/vmware-tanzu/sonobuoy/pkg/client"
	sonodynamic "github.com/vmware-tanzu/sonobuoy/pkg/dynamic"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

var (
	spinnerType     int           = 14
	spinnerDuration time.Duration = 2000 * time.Millisecond
	spinnerColor                  = "red"
)

func getSonobuoyClient(cfg *rest.Config) (*client.SonobuoyClient, error) {
	var skc *sonodynamic.APIHelper
	var err error
	if cfg != nil {
		skc, err = sonodynamic.NewAPIHelperFromRESTConfig(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "couldn't get sonobuoy api helper")
		}
	}
	return client.NewSonobuoyClient(cfg, skc)
}

func getSonobuoyClientFromKubecfg(kubecfg Kubeconfig) (*client.SonobuoyClient, error) {
	cfg, err := kubecfg.Get()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rest config")
	}
	return getSonobuoyClient(cfg)
}
