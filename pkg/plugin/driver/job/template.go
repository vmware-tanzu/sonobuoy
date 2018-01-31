package job

import (
	"github.com/heptio/sonobuoy/pkg/templates"
)

var jobTemplate = templates.NewTemplate("jobTemplate", `
---
apiVersion: v1
kind: Pod
metadata:
  annotations:
    sonobuoy-driver: Job
    sonobuoy-plugin: {{.PluginName}}
    sonobuoy-result-type: {{.ResultType}}
  labels:
    component: sonobuoy
    sonobuoy-run: '{{.SessionID}}'
    tier: analysis
  name: sonobuoy-{{.PluginName}}-job-{{.SessionID}}
  namespace: '{{.Namespace}}'
spec:
  containers:
  - {{.ProducerContainer | indent 4}}
  - command:
    - sh
    - -c
    - /sonobuoy worker global -v 5 --logtostderr
    env:
    - name: NODE_NAME
      valueFrom:
        fieldRef:
          fieldPath: spec.nodeName
    - name: RESULTS_DIR
      value: /tmp/results
    - name: MASTER_URL
      value: '{{.MasterAddress}}'
    - name: RESULT_TYPE
      value: {{.ResultType}}
    - name: CA_CERT
      value: |
        {{.CACert}}
    image: gcr.io/heptio-images/sonobuoy:master
    imagePullPolicy: Always
    name: sonobuoy-worker
    volumeMounts:
    - mountPath: /tmp/results
      name: results
      readOnly: false
  restartPolicy: Never
  serviceAccountName: sonobuoy-serviceaccount
  tolerations:
  - effect: NoSchedule
    key: node-role.kubernetes.io/master
    operator: Exists
  - key: CriticalAddonsOnly
    operator: Exists
  volumes:
  - emptyDir: {}
    name: results
`)
