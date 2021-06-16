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

package plugin

const (
	// GracefulShutdownPeriod is how long plugins have to cleanly finish before they are terminated.
	GracefulShutdownPeriod = 60

	// ResultsDir is the directory where results will be available in Sonobuoy plugin containers.
	ResultsDir = "/tmp/results"

	// TimeoutErrMsg is the message used when Sonobuoy experiences a timeout while waiting for results.
	TimeoutErrMsg = "Plugin timeout while waiting for results so there are no results. Check pod logs or other cluster details for more information as to why this occurred."
)
