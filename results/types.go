package results

import (
	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/discovery"
)

// Metadata is all the stuff about the Sonobuoy run and how long it took to query the system.
type Metadata struct {
	// Config is the config used during this Sonobuoy run.
	Config config.Config

	// QueryMetadata shows how long each object took to query.
	QueryData []discovery.QueryData
}
