/*
Copyright 2017 The Kubernetes Authors.
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

// NOTE: This is manually replicated from: https://github.com/kubernetes/kubernetes/blob/master/test/utils/image/manifest.go

package image

func (r *RegistryList) v1_13() map[string]Config {

	e2eRegistry := r.E2eRegistry
	dockerLibraryRegistry := r.DockerLibraryRegistry
	sampleRegistry := r.SampleRegistry
	etcdRegistry := r.EtcdRegistry
	gcRegistry := r.GcRegistry

	configs := map[string]Config{}
	configs["CRDConversionWebhook"] = Config{e2eRegistry, "crd-conversion-webhook", "1.13rev2"}
	configs["AdmissionWebhook"] = Config{e2eRegistry, "webhook", "1.14v1"}
	configs["APIServer"] = Config{e2eRegistry, "sample-apiserver", "1.10"}
	configs["AppArmorLoader"] = Config{e2eRegistry, "apparmor-loader", "1.0"}
	configs["AuditProxy"] = Config{e2eRegistry, "audit-proxy", "1.0"}
	configs["BusyBox"] = Config{dockerLibraryRegistry, "busybox", "1.29"}
	configs["CheckMetadataConcealment"] = Config{e2eRegistry, "metadata-concealment", "1.2"}
	configs["CudaVectorAdd"] = Config{e2eRegistry, "cuda-vector-add", "1.0"}
	configs["CudaVectorAdd2"] = Config{e2eRegistry, "cuda-vector-add", "2.0"}
	configs["Dnsutils"] = Config{e2eRegistry, "dnsutils", "1.1"}
	configs["EchoServer"] = Config{e2eRegistry, "echoserver", "2.2"}
	configs["EntrypointTester"] = Config{e2eRegistry, "entrypoint-tester", "1.0"}
	configs["Etcd"] = Config{etcdRegistry, "etcd", "v3.3.10"}
	configs["Fakegitserver"] = Config{e2eRegistry, "fakegitserver", "1.0"}
	configs["GBFrontend"] = Config{sampleRegistry, "gb-frontend", "v6"}
	configs["GBRedisSlave"] = Config{sampleRegistry, "gb-redisslave", "v3"}
	configs["Hostexec"] = Config{e2eRegistry, "hostexec", "1.1"}
	configs["IpcUtils"] = Config{e2eRegistry, "ipc-utils", "1.0"}
	configs["Iperf"] = Config{e2eRegistry, "iperf", "1.0"}
	configs["JessieDnsutils"] = Config{e2eRegistry, "jessie-dnsutils", "1.0"}
	configs["Kitten"] = Config{e2eRegistry, "kitten", "1.0"}
	configs["Liveness"] = Config{e2eRegistry, "liveness", "1.0"}
	configs["LogsGenerator"] = Config{e2eRegistry, "logs-generator", "1.0"}
	configs["Mounttest"] = Config{e2eRegistry, "mounttest", "1.0"}
	configs["MounttestUser"] = Config{e2eRegistry, "mounttest-user", "1.0"}
	configs["Nautilus"] = Config{e2eRegistry, "nautilus", "1.0"}
	configs["Net"] = Config{e2eRegistry, "net", "1.0"}
	configs["Netexec"] = Config{e2eRegistry, "netexec", "1.1"}
	configs["Nettest"] = Config{e2eRegistry, "nettest", "1.0"}
	configs["Nginx"] = Config{dockerLibraryRegistry, "nginx", "1.14-alpine"}
	configs["NginxNew"] = Config{dockerLibraryRegistry, "nginx", "1.15-alpine"}
	configs["Nonewprivs"] = Config{e2eRegistry, "nonewprivs", "1.0"}
	configs["NoSnatTest"] = Config{e2eRegistry, "no-snat-test", "1.0"}
	configs["NoSnatTestProxy"] = Config{e2eRegistry, "no-snat-test-proxy", "1.0"}
	// Pause - when these values are updated, also update cmd/kubelet/app/options/container_runtime.go
	configs["Pause"] = Config{gcRegistry, "pause", "3.1"}
	configs["Porter"] = Config{e2eRegistry, "porter", "1.0"}
	configs["PortForwardTester"] = Config{e2eRegistry, "port-forward-tester", "1.0"}
	configs["Redis"] = Config{e2eRegistry, "redis", "1.0"}
	configs["ResourceConsumer"] = Config{e2eRegistry, "resource-consumer", "1.5"}
	configs["ResourceController"] = Config{e2eRegistry, "resource-consumer/controller", "1.0"}
	configs["ServeHostname"] = Config{e2eRegistry, "serve-hostname", "1.1"}
	configs["TestWebserver"] = Config{e2eRegistry, "test-webserver", "1.0"}
	configs["VolumeNFSServer"] = Config{e2eRegistry, "volume/nfs", "1.0"}
	configs["VolumeISCSIServer"] = Config{e2eRegistry, "volume/iscsi", "1.0"}
	configs["VolumeGlusterServer"] = Config{e2eRegistry, "volume/gluster", "1.0"}
	configs["VolumeRBDServer"] = Config{e2eRegistry, "volume/rbd", "1.0.1"}
	return configs
}
