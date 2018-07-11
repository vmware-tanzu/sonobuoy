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

var (
	//ErrImageVersionNoClient is the error returned when we need a client but didn't get on
	ErrImageVersionNoClient = errors.New(`can't use nil client with "auto" image version`)
)

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
	switch str {
	case ConformanceImageVersionAuto:
		*c = ConformanceImageVersionAuto
	case ConformanceImageVersionLatest:
		*c = ConformanceImageVersionLatest
	default:
		if err := validateVersion(str); err != nil {
			return err
		}
		*c = ConformanceImageVersion(str)
	}

	return nil
}

// Get retrieves the preset version if there is one, or queries client if the ConformanceImageVersion is set to `auto`.
// kubernetes.Interface.Discovery() provides ServerVersionInterface.
func (c *ConformanceImageVersion) Get(client discovery.ServerVersionInterface) (string, error) {
	if *c == ConformanceImageVersionAuto {
		if client == nil {
			return "", ErrImageVersionNoClient
		}
		version, err := client.ServerVersion()
		if err != nil {
			return "", errors.Wrap(err, "couldn't retrieve server version")
		}

		if err := validateVersion(version.GitVersion); err != nil {
			return "", err
		}

		return version.GitVersion, nil
	}
	return string(*c), nil
}

func validateVersion(v string) error {
	version, err := version.NewVersion(v)
	if err == nil {
		if version.Metadata() != "" || version.Prerelease() != "" {
			err = errors.New("version cannot have prelease or metadata, please use a stable version")
		} else if !strings.HasPrefix(v, "v") {
			err = errors.New("version must start with v")
		}
	}
	return errors.Wrapf(err, "version %q is invalid", v)
}
