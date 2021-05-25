/*
Copyright the Sonobuoy contributors 2019

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
	"bytes"
	"flag"
	"io/ioutil"
	"testing"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/manifest"
	v1 "k8s.io/api/core/v1"
)

var update = flag.Bool("update", false, "update .golden files")

func TestPluginGenDef(t *testing.T) {
	testCases := []struct {
		desc       string
		cfg        GenPluginDefConfig
		expectFile string
		expectErr  string
	}{
		{
			desc: "Container fields",
			cfg: GenPluginDefConfig{
				def: manifest.Manifest{
					Spec: manifest.Container{
						Container: v1.Container{
							Name:    "n",
							Image:   "img",
							Command: []string{"/bin/sh", "-c", "./run.sh"},
						},
					},
				},
			},
			expectFile: "testdata/pluginDef-container.golden",
		}, {
			desc: "Env vars",
			cfg: GenPluginDefConfig{
				def: manifest.Manifest{},
				env: map[string]string{"FOO": "bar"},
			},
			expectFile: "testdata/pluginDef-env.golden",
		}, {
			desc: "Sonobuoy config",
			cfg: GenPluginDefConfig{
				def: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						Driver:     "overridden by validated value",
						PluginName: "n",
					},
				},
				driver: "Job",
			},
			expectFile: "testdata/pluginDef-sonoconfig.golden",
		}, {
			// The serialization is really handled by go-yaml/yaml so this
			// test is mainly just for doc/sanity check. The rules for YAML
			// quoting are complex and I'm glad its not our responsibility.
			desc: "Strings with special chars should be quoted",
			cfg: GenPluginDefConfig{
				def: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						Driver:     "overridden by validated value",
						PluginName: "n",
					},
					Spec: manifest.Container{
						Container: v1.Container{
							Name:    "n - foo",
							Image:   "img:v1",
							Command: []string{"/bin/sh", "- c", "./run.sh"},
						},
					},
				},
				env: map[string]string{"FOO": "- bar"},
			},
			expectFile: "testdata/pluginDef-quotes.golden",
		}, {
			desc: "default PodSpec is included if requested",
			cfg: GenPluginDefConfig{
				showDefaultPodSpec: true,
				def: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName: "n",
					},
					Spec: manifest.Container{},
				},
			},
			expectFile: "testdata/pluginDef-default-podspec.golden",
		}, {
			desc: "nodeSelectors works if default PodSpec is requested",
			cfg: GenPluginDefConfig{
				showDefaultPodSpec: true,
				nodeSelector:       map[string]string{"foo": "bar", "kubernetes.io/os": "windows"},
				def: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName: "n",
					},
					Spec: manifest.Container{},
				},
			},
			expectFile: "testdata/pluginDef-nodeselector-default-podspec.golden",
		}, {
			desc: "nodeSelectors works if default PodSpec is not requested",
			cfg: GenPluginDefConfig{
				showDefaultPodSpec: true,
				nodeSelector:       map[string]string{"foo": "bar", "kubernetes.io/os": "windows"},
				def: manifest.Manifest{
					SonobuoyConfig: manifest.SonobuoyConfig{
						PluginName: "n",
					},
					Spec: manifest.Container{},
				},
			},
			expectFile: "testdata/pluginDef-nodeselector.golden",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			manifest, err := genPluginDef(&tC.cfg)
			if err != nil {
				if len(tC.expectErr) == 0 {
					t.Fatalf("Expected nil error but got %v", err)
				}
				if err.Error() != tC.expectErr {
					t.Fatalf("Expected error %q but got %q", err, tC.expectErr)
				}
			}

			if *update {
				ioutil.WriteFile(tC.expectFile, manifest, 0666)
			} else {
				fileData, err := ioutil.ReadFile(tC.expectFile)
				if err != nil {
					t.Fatalf("Failed to read golden file %v: %v", tC.expectFile, err)
				}
				if !bytes.Equal(fileData, manifest) {
					t.Errorf("Expected manifest to equal goldenfile: %v but instead got:\n\n%v", tC.expectFile, string(manifest))
				}
			}
		})
	}
}
