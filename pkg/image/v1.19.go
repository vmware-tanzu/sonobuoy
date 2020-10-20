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

// NOTE: This is manually replicated from: https://github.com/kubernetes/kubernetes/blob/v1.18.0/test/utils/image/manifest.go#L208-L248

package image

func (r *RegistryList) v1_19() map[int]Config {

	buildImageRegistry := r.BuildImageRegistry
	e2eRegistry := r.E2eRegistry
	dockerLibraryRegistry := r.DockerLibraryRegistry
	gcRegistry := r.GcRegistry
	gcAuthenticatedRegistry := r.GcAuthenticatedRegistry
	invalidRegistry := r.InvalidRegistry
	privateRegistry := r.PrivateRegistry
	dockerGluster := r.DockerGluster
	promoterE2eRegistry := r.PromoterE2eRegistry
	sigStorageRegistry := r.SigStorageRegistry
	e2eVolumeRegistry := r.E2eVolumeRegistry

	configs := map[int]Config{}
	configs[Agnhost] = Config{promoterE2eRegistry, "agnhost", "2.20"}
	configs[AgnhostPrivate] = Config{privateRegistry, "agnhost", "2.6"}
	configs[AuthenticatedAlpine] = Config{gcAuthenticatedRegistry, "alpine", "3.7"}
	configs[AuthenticatedWindowsNanoServer] = Config{gcAuthenticatedRegistry, "windows-nanoserver", "v1"}
	configs[APIServer] = Config{e2eRegistry, "sample-apiserver", "1.17"}
	configs[AppArmorLoader] = Config{e2eRegistry, "apparmor-loader", "1.0"}
	configs[BusyBox] = Config{dockerLibraryRegistry, "busybox", "1.29"}
	configs[CheckMetadataConcealment] = Config{e2eRegistry, "metadata-concealment", "1.2"}
	configs[CudaVectorAdd] = Config{e2eRegistry, "cuda-vector-add", "1.0"}
	configs[CudaVectorAdd2] = Config{e2eRegistry, "cuda-vector-add", "2.0"}
	configs[DebianIptables] = Config{buildImageRegistry, "debian-iptables", "v12.1.2"}
	configs[EchoServer] = Config{e2eRegistry, "echoserver", "2.2"}
	configs[Etcd] = Config{gcRegistry, "etcd", "3.4.9"}
	configs[GlusterDynamicProvisioner] = Config{dockerGluster, "glusterdynamic-provisioner", "v1.0"}
	configs[Httpd] = Config{dockerLibraryRegistry, "httpd", "2.4.38-alpine"}
	configs[HttpdNew] = Config{dockerLibraryRegistry, "httpd", "2.4.39-alpine"}
	configs[InvalidRegistryImage] = Config{invalidRegistry, "alpine", "3.1"}
	configs[IpcUtils] = Config{e2eRegistry, "ipc-utils", "1.0"}
	configs[JessieDnsutils] = Config{e2eRegistry, "jessie-dnsutils", "1.0"}
	configs[Kitten] = Config{e2eRegistry, "kitten", "1.0"}
	configs[Nautilus] = Config{e2eRegistry, "nautilus", "1.0"}
	configs[NFSProvisioner] = Config{sigStorageRegistry, "nfs-provisioner", "v2.2.2"}
	configs[Nginx] = Config{dockerLibraryRegistry, "nginx", "1.14-alpine"}
	configs[NginxNew] = Config{dockerLibraryRegistry, "nginx", "1.15-alpine"}
	configs[Nonewprivs] = Config{e2eRegistry, "nonewprivs", "1.0"}
	configs[NonRoot] = Config{e2eRegistry, "nonroot", "1.0"}
	// Pause - when these values are updated, also update cmd/kubelet/app/options/container_runtime.go
	configs[Pause] = Config{gcRegistry, "pause", "3.2"}
	configs[Perl] = Config{dockerLibraryRegistry, "perl", "5.26"}
	configs[PrometheusDummyExporter] = Config{gcRegistry, "prometheus-dummy-exporter", "v0.1.0"}
	configs[PrometheusToSd] = Config{gcRegistry, "prometheus-to-sd", "v0.5.0"}
	configs[Redis] = Config{dockerLibraryRegistry, "redis", "5.0.5-alpine"}
	configs[RegressionIssue74839] = Config{e2eRegistry, "regression-issue-74839-amd64", "1.0"}
	configs[ResourceConsumer] = Config{e2eRegistry, "resource-consumer", "1.5"}
	configs[SdDummyExporter] = Config{gcRegistry, "sd-dummy-exporter", "v0.2.0"}
	configs[VolumeNFSServer] = Config{e2eVolumeRegistry, "nfs", "1.0"}
	configs[VolumeISCSIServer] = Config{e2eVolumeRegistry, "iscsi", "2.0"}
	configs[VolumeGlusterServer] = Config{e2eVolumeRegistry, "gluster", "1.0"}
	configs[VolumeRBDServer] = Config{e2eVolumeRegistry, "rbd", "1.0.1"}

	return configs
}
