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

// Set the ImageVersion to either the string "auto" or a version string. The resulting version string
// will be forced into semver version with a 'v' prefix. You can set pre-release/metadata but it will
// fill in the minor/patch values as '0' if missing. E.g. 1+beta would yield v1.0.0+beta
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

		// It is important to set the string with the `v` prefix in order
		// to be consistent with server version reporting and image tagging norms.
		*c = ConformanceImageVersion(fmt.Sprintf("v%v", version.String()))
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

		return conformanceTagFromSemver(version.GitVersion)
	}
	return string(*c), nil
}

// conformanceTagFromSemver uses the gitversion to choose the proper conformance image to use.
// Prereleases are considered, but metadata and provider-specific info is discarded.
func conformanceTagFromSemver(gitVersion string) (string, error) {
	parsedVersion, err := validateVersion(gitVersion)
	if err != nil {
		return "", err
	}

	segments := parsedVersion.Segments()
	if len(segments) < 2 {
		return "", fmt.Errorf("version %q only has %d segments, need at least 2", gitVersion, len(segments))
	}

	// Not sure that this would be hit but default to adding the last
	// segment as 0 per convention (upstream + semver).
	if len(segments) < 3 {
		return fmt.Sprintf("v%d.%d.%d", segments[0], segments[1], 0), nil
	}

	// Upstream Kubernetes publishes the conformance images for prereleases as well; we should use them
	// to ease testing new versions. Some vendors seem to put their name as prerelease instead of
	// build metadata so handle on a case-by-case basis.
	switch pr := parsedVersion.Prerelease(); {
	case strings.HasPrefix(pr, "rc"),
		strings.HasPrefix(pr, "alpha"),
		strings.HasPrefix(pr, "beta"):
		return fmt.Sprintf("v%d.%d.%d-%v", segments[0], segments[1], segments[2], parsedVersion.Prerelease()), nil
	}
	return fmt.Sprintf("v%d.%d.%d", segments[0], segments[1], segments[2]), nil
}

func validateVersion(v string) (*version.Version, error) {
	version, err := version.NewVersion(v)
	if err == nil {
		if !strings.HasPrefix(v, "v") {
			err = errors.New("version must start with v")
		}
	}

	return version, errors.Wrapf(err, "version %q is invalid", v)
}
