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
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/templates"
)

// templateValues are used for direct template substitution for manifest generation.
type templateValues struct {
	E2EFocus       string
	SonobuoyConfig string
	SonobuoyImage  string
	Version        string
	Namespace      string
}

func (c *SonobuoyClient) GenerateManifest(cfg *GenConfig) ([]byte, error) {
	mode := cfg.ModeName.Get()
	if mode == nil {
		return nil, fmt.Errorf("unknown mode: %q", cfg.ModeName.String())
	}

	cfg.Config.WorkerImage = cfg.Image

	marshalledConfig, err := json.Marshal(cfg.Config)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't marshall selector")
	}

	tmplVals := &templateValues{
		E2EFocus:       mode.E2EFocus,
		SonobuoyConfig: string(marshalledConfig),
		SonobuoyImage:  cfg.Image,
		Version:        buildinfo.Version,
		Namespace:      cfg.Namespace,
	}

	var buf bytes.Buffer

	if err := templates.Manifest.Execute(&buf, tmplVals); err != nil {
		return nil, errors.Wrap(err, "couldn't execute manifest template")
	}

	return buf.Bytes(), nil
}
