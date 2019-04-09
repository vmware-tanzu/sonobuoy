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

package image

import (
	"fmt"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
		version, err := validateVersion(str)
		if err != nil {
			return err
		}
		*c = ConformanceImageVersion(version.String())
	}

	return nil
}

// Get retrieves the preset version if there is one, or queries client if the ConformanceImageVersion is set to `auto`.
// kubernetes.Interface.Discovery() provides ServerVersionInterface.
// Don't require the entire kubernetes.Interface to simplify the required test mocks
func (c *ConformanceImageVersion) Get(client discovery.ServerVersionInterface) (string, error) {
	if *c == ConformanceImageVersionAuto {
		if client == nil {
			return "", ErrImageVersionNoClient
		}
		version, err := client.ServerVersion()
		if err != nil {
			return "", errors.Wrap(err, "couldn't retrieve server version")
		}

		parsedVersion, err := validateVersion(version.GitVersion)
		if err != nil {
			return "", err
		}

		segments := parsedVersion.Segments()
		if len(segments) < 2 {
			return "", fmt.Errorf("version %q only has %d segments, need at least 2", version.GitVersion, len(segments))
		}

		// Temporary logic in place to truncate auto-resolved versions while we
		// transition to upstream. If < 1.14 return 2 segments due to lag behind
		// releases. Otherwise return 3. Use the segments instead of .major and
		// .minor because GKE's .minor is `10+` instead of `10`.
		if segments[0] == 1 && segments[1] < 14 {
			return fmt.Sprintf("v%d.%d", segments[0], segments[1]), nil
		}

		// Not sure that this would be hit but default to adding the last
		// segment as 0 per convention (upstream + semver).
		if len(segments) < 3 {
			return fmt.Sprintf("v%d.%d.%d", segments[0], segments[1], 0), nil
		}
		return fmt.Sprintf("v%d.%d.%d", segments[0], segments[1], segments[2]), nil
	}
	return string(*c), nil
}

func validateVersion(v string) (*version.Version, error) {
	version, err := version.NewVersion(v)
	if err == nil {
		if version.Metadata() != "" || version.Prerelease() != "" {
			logrus.Warningf("Version %v is not a stable version, conformance image may not exist upstream", v)
		} else if !strings.HasPrefix(v, "v") {
			err = errors.New("version must start with v")
		}
	}

	return version, errors.Wrapf(err, "version %q is invalid", v)
}
