package app

import (
	"github.com/heptio/sonobuoy/pkg/client"
	sonodynamic "github.com/heptio/sonobuoy/pkg/dynamic"

	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

func getSonobuoyClient(cfg *rest.Config) (*client.SonobuoyClient, error) {
	skc, err := sonodynamic.NewAPIHelperFromRESTConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get sonobuoy api helper")
	}
	return client.NewSonobuoyClient(cfg, skc)
}
