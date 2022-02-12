/*
Copyright the Sonobuoy contributors 2022

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

package discovery

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/sonobuoy/pkg/client/results"
	v1 "k8s.io/api/core/v1"
)

type ClusterSummary struct {
	NodeHealth HealthInfo `json:"node_health" yaml:"node_health"`
	PodHealth  HealthInfo `json:"pod_health" yaml:"pod_health"`
	APIVersion string     `json:"api_version" yaml:"api_version"`
	ErrorInfo  LogSummary `json:"error_summary" yaml:"error_summary"`
}

type HealthInfo struct {
	Total   int                 `json:"total_nodes" yaml:"total_nodes"`
	Healthy int                 `json:"healthy_nodes" yaml:"healthy_nodes"`
	Details []HealthInfoDetails `json:"details,omitempty" yaml:"details,omitempty"`
}

type HealthInfoDetails struct {
	Name      string `json:"name" yaml:"name"`
	Healthy   bool   `json:"healthy" yaml:"healthy"`
	Ready     string `json:"ready" yaml:"ready"`
	Reason    string `json:"reason,omitempty" yaml:"reason,omitempty"`
	Message   string `json:"message,omitempty" yaml:"message,omitempty"`
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
}

// ReadPodHealth lists all the directories in path.Join(tarballRootDir, NSResourceLocation)
// In each, check if it contains the file CorePodFile, if so, read each as v1.PodList,
// And loop through _, pod := range podList.Items,
// and scan _, condition := range pod.Status.Conditions
// and check if condition.Status == v1.ConditionTrue:
// for each of these, if they are false, add the condition.Reason and condition.Message to a string
func ReadPodHealth(r *results.Reader) (HealthInfo, error) {
	health := HealthInfo{}
	health.Details = make([]HealthInfoDetails, 0)

	findAndProcessPodCoreFiles := func(filePath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == results.CorePodsFile {
			podList := &v1.PodList{}
			err := results.ExtractFileIntoStruct(filePath, filePath, info, &podList)
			if err != nil {
				logrus.Errorf("Failed to read pod health information from file '%s': %s", filePath, err)
				logrus.Errorf("File '%s' will be skipped", filePath)
				return filepath.SkipDir
			}
			health.Total += len(podList.Items)

			for _, pod := range podList.Items {
				podHealth := HealthInfoDetails{}
				podHealth.Healthy = pod.Status.Phase == v1.PodRunning || pod.Status.Phase == v1.PodSucceeded
				podHealth.Name = pod.ObjectMeta.Name
				podHealth.Namespace = pod.ObjectMeta.Namespace
				if !podHealth.Healthy {
					//scan pod.Conditions, and find the first where condition.Status != v1.ConditionTrue
					for _, condition := range pod.Status.Conditions {
						if condition.Status != v1.ConditionTrue {
							podHealth.Ready = string(condition.Status)
							podHealth.Reason = condition.Reason
							podHealth.Message = condition.Message
							//We don't need to look further
							break
						}
					}
				} else {
					//Otherwise count this pod as healthy
					health.Healthy++
					//And fill in the remaining fields
					podHealth.Ready = string(v1.PodReady)
				}
				health.Details = append(health.Details, podHealth)
			}

			//We can skip the current directory anyway
			return filepath.SkipDir
		}
		return nil
	}

	if err := r.WalkFiles(findAndProcessPodCoreFiles); err != nil {
		logrus.Errorf("Failed to load pod health: %s", err)
		return health, err
	}
	return health, nil
}

//ReadHealthSummary reads the core_v1_nodes.json file from ClusterResourceLocation
//and returns a summary of the health fo the cluster, ready to be saved
//tarballRootDir is the directory that will be used to provide the contents of the tarball
func ReadHealthSummary(tarballRootDir string) (ClusterSummary, error) {
	summary := ClusterSummary{}
	nodes := &v1.NodeList{}
	r := results.NewReaderFromDir(tarballRootDir)
	fileName := path.Join(ClusterResourceLocation, results.CoreNodesFile)
	err := r.WalkFiles(func(path string, info os.FileInfo, err error) error {
		return results.ExtractFileIntoStruct(fileName, path, info, &nodes)
	})
	if err != nil {
		logrus.Errorf("Failed to read health summary: failed to read the node list from '%s': %s", fileName, err)
		return summary, err
	}
	summary.NodeHealth.Total = len(nodes.Items)
	summary.NodeHealth.Details = make([]HealthInfoDetails, summary.NodeHealth.Total)

	for nodeIdx, node := range nodes.Items {
		summary.NodeHealth.Details[nodeIdx].Name = node.ObjectMeta.Name
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady {
				summary.NodeHealth.Details[nodeIdx].Healthy = condition.Status == v1.ConditionTrue
				summary.NodeHealth.Details[nodeIdx].Ready = string(condition.Status)
				summary.NodeHealth.Details[nodeIdx].Reason = condition.Reason
				summary.NodeHealth.Details[nodeIdx].Message = condition.Message
				//And count the healthy nodes
				if summary.NodeHealth.Details[nodeIdx].Healthy {
					summary.NodeHealth.Healthy++
				}
			}
		}
	}
	summary.APIVersion, _ = r.ReadVersion()
	//ReadVersion already logged this error, and we can continue with the rest of the information

	summary.PodHealth, _ = ReadPodHealth(r)
	//ReadPodHealth already logged this error, and we can continue with the rest of the information

	summary.ErrorInfo, _ = ReadLogSummaryWithDefaultPatterns(r)
	//ReadLogSummary already logged this error, and we can continue with the rest of the information

	return summary, nil
}

// SaveHealthSummary loads data from
// - CoreNodesFile in ClusterResourceLocation in tarballRootDir
// - ServerVersionFile in tarballRootDir
// - CorePodsFile in NSResourceLocation in tarballRootDir
// Extract health information, and saves the result as json to
// results.ClusterHealthFilePath() in tarballRootDir
// SaveHealthSummary assumes that all the directories including MetaLocation have already been created
func SaveHealthSummary(tarballRootDir string) error {
	outputFileName := path.Join(tarballRootDir, results.ClusterHealthFilePath())
	healthSummary, err := ReadHealthSummary(tarballRootDir)
	if err != nil {
		logrus.Errorf("Failed to read cluster health information from '%s': %s", tarballRootDir, err)
		logrus.Errorf("File '%s' will not be included in '%s'.", outputFileName, tarballRootDir)
		return err
	}
	data, err := json.Marshal(healthSummary)
	if err != nil {
		logrus.Errorf("Failed to marshall health information to json: %s", err)
		logrus.Errorf("File '%s' will not be included in '%s'.", outputFileName, tarballRootDir)
		return err
	}
	err = ioutil.WriteFile(outputFileName, data, os.FileMode(0644))
	if err != nil {
		logrus.Errorf("Failed to write health information to file '%s': %s", outputFileName, err)
		return err
	}
	return nil

}
