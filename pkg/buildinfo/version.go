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

// Package buildinfo holds build-time information like the sonobuoy version.
// This is a separate package so that other packages can import it without
// worrying about introducing circular dependencies.
package buildinfo

// Version is the current version of Sonobuoy, set by the go linker's -X flag at build time
var Version = "v0.52.0"

// GitSHA is the actual commit that is being built, set by the go linker's -X flag at build time.
var GitSHA string

// MinimumKubeVersion is the lowest API version of Kubernetes this release of Sonobuoy supports.
// (johnschnake): Not sure that we are really updating this anymore; now that we are generally
// independant  of k8s releases, we don't have to bump this if we don't have a clear reason.
var MinimumKubeVersion = "1.17.0"

// MaximumKubeVersion is the highest API version of Kubernetes this release of Sonobuoy supports.
// Set to 1.99.99 so as not to error/warn on new versions for development.
var MaximumKubeVersion = "1.99.99"
