/*
Copyright 2019 Sonobuoy contributors 2019

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

package manifest

import (
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/driver"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"

	"github.com/pkg/errors"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

// ToYAML will serialize the manifest and add the default podspec (based on the appropriate drive)
// if not already set in the manifest.
func ToYAML(m *manifest.Manifest, showDefaultPodSpec bool) ([]byte, error) {
	if showDefaultPodSpec && m.PodSpec == nil {
		m.PodSpec = &manifest.PodSpec{
			PodSpec: driver.DefaultPodSpec(m.SonobuoyConfig.Driver),
		}
	}
	yaml, err := kuberuntime.Encode(manifest.Encoder, m)
	return yaml, errors.Wrapf(err, "serializing plugin %v as YAML", m.SonobuoyConfig.PluginName)
}
