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

package operations

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/heptio/sonobuoy/cmd/sonobuoy/app/args"
	"github.com/heptio/sonobuoy/pkg/buildinfo"
	"github.com/heptio/sonobuoy/pkg/templates"
)

// GenConfig are the input options for running
// TODO: Figure out chained subcommands or how to share input options from RunConfig
type GenConfig struct {
	ModeName  args.Mode
	Image     args.SonobuoyImage
	Namespace args.Namespace
}

type templateValues struct {
	E2EFocus       string
	PluginSelector string
	SonobuoyImage  string
	Version        string
	Namespace      string
}

// GenerateManifest fills in a template with a Sonobuoy config
func GenerateManifest(cfg GenConfig) ([]byte, error) {
	mode := cfg.ModeName.Get()
	if mode == nil {
		return nil, fmt.Errorf("unknown mode: %q", cfg.ModeName.String())
	}
	marshalledSelector, err := json.Marshal(mode.Selectors)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't marshall selector")
	}

	tmplVals := &templateValues{
		E2EFocus:       mode.E2EFocus,
		PluginSelector: string(marshalledSelector),
		SonobuoyImage:  cfg.Image.Get(),
		Version:        buildinfo.Version,
		Namespace:      cfg.Namespace.Get(),
	}

	var buf bytes.Buffer

	if err := templates.Manifest.Execute(&buf, tmplVals); err != nil {
		return nil, errors.Wrap(err, "couldn't execute manifest template")
	}

	return buf.Bytes(), nil
}
