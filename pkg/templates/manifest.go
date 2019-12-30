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

package templates

// Manifest is the template found in examples
var Manifest = NewTemplate("manifest", `
---
apiVersion: v1
kind: Namespace
metadata:
  name: {{.Namespace}}
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-serviceaccount
  namespace: {{.Namespace}}
{{- if .EnableRBAC }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  labels:
    component: sonobuoy
    namespace: {{.Namespace}}
  name: sonobuoy-serviceaccount-{{.Namespace}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: sonobuoy-serviceaccount-{{.Namespace}}
subjects:
- kind: ServiceAccount
  name: sonobuoy-serviceaccount
  namespace: {{.Namespace}}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    component: sonobuoy
    namespace: {{.Namespace}}
  name: sonobuoy-serviceaccount-{{.Namespace}}
rules:
- apiGroups:
  - '*'
  resources:
  - '*'
  verbs:
  - '*'
- nonResourceURLs:
  - '/metrics'
  - '/logs'
  - '/logs/*'
  verbs:
  - 'get'
{{- end }}
{{- if .SSHKey }}
---
apiVersion: v1
kind: Secret
metadata:
  name: ssh-key
  namespace: {{.Namespace}}
type: Opaque
data:
  id_rsa: {{.SSHKey}}
{{- end }}
---
apiVersion: v1
data:
  config.json: |
    {{.SonobuoyConfig}}
kind: ConfigMap
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-config-cm
  namespace: {{.Namespace}}
---
apiVersion: v1
{{- if .Plugins }}
data:{{- range $i, $v := .Plugins }}
  plugin-{{- $i -}}.yaml: |
    {{ indent 4 $v }}
{{- end }}
{{- end }}
kind: ConfigMap
metadata:
  labels:
    component: sonobuoy
  name: sonobuoy-plugins-cm
  namespace: {{.Namespace}}
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    component: sonobuoy
    run: sonobuoy-master
    tier: analysis
  name: sonobuoy
  namespace: {{.Namespace}}
{{- if .CustomAnnotations }}
  annotations:{{- range $k, $v := .CustomAnnotations }}
    {{ indent 4 $k}}: {{$v}}
{{- end }}
{{- end }}
spec:
  containers:
  - command:
    - /bin/bash
    - -c
    - /sonobuoy master --no-exit=true -v 3 --logtostderr
    env:
    - name: SONOBUOY_ADVERTISE_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    image: {{.SonobuoyImage}}
    imagePullPolicy: {{.ImagePullPolicy}}
    name: kube-sonobuoy
    volumeMounts:
    - mountPath: /etc/sonobuoy
      name: sonobuoy-config-volume
    - mountPath: /plugins.d
      name: sonobuoy-plugins-volume
    - mountPath: /tmp/sonobuoy
      name: output-volume
  {{- if .ImagePullSecrets }}
  imagePullSecrets:
  - name: {{.ImagePullSecrets}}
  {{- end }}
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  tolerations:
  - key: "kubernetes.io/e2e-evict-taint-key"
    operator: "Exists"
  volumes:
  - configMap:
      name: sonobuoy-config-cm
    name: sonobuoy-config-volume
  - configMap:
      name: sonobuoy-plugins-cm
    name: sonobuoy-plugins-volume
  - emptyDir: {}
    name: output-volume
---
{{- if .CustomRegistries }}
apiVersion: v1
data:
  repo-list.yaml: |
    {{ indent 4 .CustomRegistries }}
kind: ConfigMap
metadata:
  name: repolist-cm
  namespace: {{.Namespace}}
---
{{- end}}
apiVersion: v1
kind: Service
metadata:
  labels:
    component: sonobuoy
    run: sonobuoy-master
  name: sonobuoy-master
  namespace: {{.Namespace}}
spec:
  ports:
  - port: 8080
    protocol: TCP
    targetPort: 8080
  selector:
    run: sonobuoy-master
  type: ClusterIP
`)
