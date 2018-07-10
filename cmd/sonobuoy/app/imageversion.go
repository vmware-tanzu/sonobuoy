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

package app

import (
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"k8s.io/client-go/discovery"
)

// ConformanceImageVersion represents the version of a conformance image, or "auto" to detect the version
type ConformanceImageVersion string

const (
	// ConformanceImageVersionAuto represents detecting the server's kubernetes version.
	ConformanceImageVersionAuto = "auto"
	// ConformanceImageVersionLatest represents always using the server's latest version.
	ConformanceImageVersionLatest = "latest"
)

// String needed for pflag.Value.
func (c *ConformanceImageVersion) String() string { return string(*c) }

// Type needed for pflag.Value.
func (c *ConformanceImageVersion) Type() string { return "ConformanceImageVersion" }

// Set the ImageVersion to either the string "auto" or a version string
func (c *ConformanceImageVersion) Set(str string) error {
	if str == ConformanceImageVersionAuto {
		*c = ConformanceImageVersionAuto
		return nil
	} else if str == ConformanceImageVersionLatest {
		*c = ConformanceImageVersionLatest
		return nil
	}

	version, err := version.NewVersion(str)
	if err != nil {
		return err
	}

	if version.Metadata() != "" || version.Prerelease() != "" {
		return errors.New("version cannot have prelease or metadata")
	}

	if !strings.HasPrefix(str, "v") {
		return errors.New("version must start with v")
	}

	*c = ConformanceImageVersion(str)
	return nil
}

// Get retrieves the preset version if there is one, or queries client if the ConformanceImageVersion is set to `auto`.
// kubernetes.Interface.Discovery() provides ServerVersionInterface.
func (c *ConformanceImageVersion) Get(client discovery.ServerVersionInterface) (string, error) {
	if *c == ConformanceImageVersionAuto {
		version, err := client.ServerVersion()
		if err != nil {
			return "", errors.Wrap(err, "couldn't retrieve server version")
		}
		return version.GitVersion, nil
	}
	return string(*c), nil
}
