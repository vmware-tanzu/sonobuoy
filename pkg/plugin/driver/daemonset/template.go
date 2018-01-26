package daemonset

import (
	"text/template"

	"github.com/heptio/sonobuoy/pkg/plugin/driver/utils"
)

var daemonSetTemplate = template.Must(
	template.New("jobTemplate").Funcs(utils.TemplateFuncs).Parse(`
---
apiVersion: extensions/v1beta1
kind: DaemonSet
metadata:
  annotations:
    sonobuoy-driver: DaemonSet
    sonobuoy-plugin: {{.PluginName}}
    sonobuoy-result-type: {{.ResultType}}
  labels:
    component: sonobuoy
    sonobuoy-run: '{{.SessionID}}'
    tier: analysis
  name: sonobuoy-{{.PluginName}}-daemon-set-{{.SessionID}}
  namespace: '{{.Namespace}}'
spec:
  selector:
    matchLabels:
      sonobuoy-run: '{{.SessionID}}'
  template:
    metadata:
      labels:
        component: sonobuoy
        sonobuoy-run: '{{.SessionID}}'
        tier: analysis
    spec:
      containers:
      - {{.ProducerContainer | indent 8}}
      - command:
        - sh
        - -c
        - /sonobuoy worker single-node -v 5 --logtostderr && sleep 3600
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
        image: gcr.io/heptio-images/sonobuoy:master
        imagePullPolicy: Always
        name: sonobuoy-worker
        volumeMounts:
        - mountPath: /tmp/results
          name: results
          readOnly: false
      dnsPolicy: ClusterFirstWithHostNet
      hostIPC: true
      hostNetwork: true
      hostPID: true
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - key: CriticalAddonsOnly
        operator: Exists
      volumes:
      - emptyDir: {}
        name: results
      - hostPath:
          path: /
        name: root
`))
