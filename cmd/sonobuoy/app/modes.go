/*
Copyright 2021 Sonobuoy Contributors

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
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type e2eModeOptions struct {
	name        string
	desc        string
	focus, skip string
	parallel    bool
}

const (
	// E2eModeQuick runs a single E2E test and the systemd log tests.
	E2eModeQuick string = "quick"

	// E2eModeNonDisruptiveConformance runs all of the `Conformance` E2E tests which are not marked as disuprtive and the systemd log tests.
	E2eModeNonDisruptiveConformance string = "non-disruptive-conformance"

	// E2eModeCertifiedConformance runs all of the `Conformance` E2E tests and the systemd log tests.
	E2eModeCertifiedConformance string = "certified-conformance"

	// nonDisruptiveSkipList should generally just need to skip disruptive tests since upstream
	// will disallow the other types of tests from being tagged as Conformance. However, in v1.16
	// two disruptive tests were  not marked as such, meaning we needed to specify them here to ensure
	// user workload safety. See https://github.com/kubernetes/kubernetes/issues/82663
	// and https://github.com/kubernetes/kubernetes/issues/82787
	nonDisruptiveSkipList = `\[Disruptive\]|NoExecuteTaintManager`
	conformanceFocus      = `\[Conformance\]`
	quickFocus            = "Pods should be submitted and removed"

	E2eModeConformanceLite = "conformance-lite"
)

var (
	liteSkips = []string{
		"Serial", "Slow", "Disruptive",
		"[sig-apps] StatefulSet Basic StatefulSet functionality [StatefulSetBasic] should have a working scale subresource [Conformance]",
		"[sig-network] EndpointSlice should create Endpoints and EndpointSlices for Pods matching a Service [Conformance]",
		"[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for multiple CRDs of same group and version but different kinds [Conformance]",
		"[sig-auth] ServiceAccounts ServiceAccountIssuerDiscovery should support OIDC discovery of service account issuer [Conformance]",
		"[sig-network] DNS should provide DNS for services  [Conformance]",
		"[sig-network] DNS should resolve DNS of partial qualified names for services [LinuxOnly] [Conformance]",
		"[sig-apps] Job should delete a job [Conformance]",
		"[sig-network] DNS should provide DNS for ExternalName services [Conformance]",
		"[sig-node] Variable Expansion should succeed in writing subpaths in container [Slow] [Conformance]",
		"[sig-apps] Daemon set [Serial] should rollback without unnecessary restarts [Conformance]",
		"[sig-api-machinery] Garbage collector should orphan pods created by rc if delete options say so [Conformance]",
		"[sig-network] Services should have session affinity timeout work for service with type clusterIP [LinuxOnly] [Conformance]",
		"[sig-network] Services should have session affinity timeout work for NodePort service [LinuxOnly] [Conformance]",
		"[sig-node] InitContainer [NodeConformance] should not start app containers if init containers fail on a RestartAlways pod [Conformance]",
		"[sig-apps] Daemon set [Serial] should update pod when spec was updated and update strategy is RollingUpdate [Conformance]",
		"[sig-api-machinery] CustomResourcePublishOpenAPI [Privileged:ClusterAdmin] works for multiple CRDs of same group but different versions [Conformance]",
		"[sig-apps] StatefulSet Basic StatefulSet functionality [StatefulSetBasic] Burst scaling should run to completion even with unhealthy pods [Slow] [Conformance]",
		`[sig-node] Probing container should be restarted with a exec "cat /tmp/health" liveness probe [NodeConformance] [Conformance]`,
		"[sig-network] Services should be able to switch session affinity for service with type clusterIP [LinuxOnly] [Conformance]",
		"[sig-node] Probing container with readiness probe that fails should never be ready and never restart [NodeConformance] [Conformance]",
		"[sig-api-machinery] Watchers should observe add, update, and delete watch notifications on configmaps [Conformance]",
		"[sig-scheduling] SchedulerPreemption [Serial] PriorityClass endpoints verify PriorityClass endpoints can be operated with different HTTP methods [Conformance]",
		"[sig-api-machinery] CustomResourceDefinition resources [Privileged:ClusterAdmin] Simple CustomResourceDefinition listing custom resource definition objects works  [Conformance]",
		"[sig-api-machinery] CustomResourceDefinition Watch [Privileged:ClusterAdmin] CustomResourceDefinition Watch watch on custom resource definition objects [Conformance]",
		"[sig-scheduling] SchedulerPreemption [Serial] validates basic preemption works [Conformance]",
		"[sig-storage] ConfigMap optional updates should be reflected in volume [NodeConformance] [Conformance]",
		"[sig-apps] StatefulSet Basic StatefulSet functionality [StatefulSetBasic] Scaling should happen in predictable order and halt if any stateful pod is unhealthy [Slow] [Conformance]",
		"[sig-storage] EmptyDir wrapper volumes should not cause race condition when used for configmaps [Serial] [Conformance]",
		"[sig-scheduling] SchedulerPreemption [Serial] validates lower priority pod preemption by critical pod [Conformance]",
		"[sig-storage] Projected secret optional updates should be reflected in volume [NodeConformance] [Conformance]",
		"[sig-apps] CronJob should schedule multiple jobs concurrently [Conformance]",
		"[sig-apps] CronJob should replace jobs when ReplaceConcurrent [Conformance]",
		"[sig-scheduling] SchedulerPreemption [Serial] PreemptionExecutionPath runs ReplicaSets to verify preemption running path [Conformance]",
		"[sig-apps] StatefulSet Basic StatefulSet functionality [StatefulSetBasic] should perform canary updates and phased rolling updates of template modifications [Conformance]",
		"[sig-apps] StatefulSet Basic StatefulSet functionality [StatefulSetBasic] should perform rolling updates and roll backs of template modifications [Conformance]",
		"[sig-node] Probing container should have monotonically increasing restart count [NodeConformance] [Conformance]",
		"[sig-node] Variable Expansion should verify that a failing subpath expansion can be modified during the lifecycle of a container [Slow] [Conformance]",
		`[sig-node] Probing container should *not* be restarted with a exec "cat /tmp/health" liveness probe [NodeConformance] [Conformance]`,
		"[sig-node] Probing container should *not* be restarted with a tcp:8080 liveness probe [NodeConformance] [Conformance]",
		"[sig-node] Probing container should *not* be restarted with a /healthz http liveness probe [NodeConformance] [Conformance]",
		"[sig-apps] CronJob should not schedule jobs when suspended [Slow] [Conformance]",
		"[sig-scheduling] SchedulerPredicates [Serial] validates that there exists conflict between pods with same hostPort and protocol but one using 0.0.0.0 hostIP [Conformance]",
		"[sig-apps] CronJob should not schedule new jobs when ForbidConcurrent [Slow] [Conformance]",

		`[k8s.io] Probing container should *not* be restarted with a exec "cat /tmp/health" liveness probe [NodeConformance] [Conformance]`,
		`[sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] should perform canary updates and phased rolling updates of template modifications [Conformance]`,
		`[sig-storage] ConfigMap updates should be reflected in volume [NodeConformance] [Conformance]`,
		`[sig-network] Services should be able to switch session affinity for NodePort service [LinuxOnly] [Conformance]`,
		`[k8s.io] Probing container with readiness probe that fails should never be ready and never restart [NodeConformance] [Conformance]`,
		`[sig-storage] Projected configMap optional updates should be reflected in volume [NodeConformance] [Conformance]`,
		`[k8s.io] Probing container should be restarted with a exec "cat /tmp/health" liveness probe [NodeConformance] [Conformance]`,
		`[sig-api-machinery] Garbage collector should delete RS created by deployment when not orphaning [Conformance]`,
		`[sig-api-machinery] Garbage collector should delete pods created by rc when not orphaning [Conformance]`,
		`[k8s.io] Probing container should have monotonically increasing restart count [NodeConformance] [Conformance]`,
		`[k8s.io] Probing container should *not* be restarted with a tcp:8080 liveness probe [NodeConformance] [Conformance]`,
		`[sig-api-machinery] Garbage collector should keep the rc around until all its pods are deleted if the deleteOptions says so [Conformance]`,
		`[sig-apps] StatefulSet [k8s.io] Basic StatefulSet functionality [StatefulSetBasic] should perform rolling updates and roll backs of template modifications [Conformance]`,
	}
)

// validModes is a map of the various valid modes. Name is duplicated as the key and in the e2eModeOptions itself.
var validModes = map[string]e2eModeOptions{
	E2eModeQuick: {
		name: E2eModeQuick, focus: quickFocus,
		desc: "Quick mode runs a single test to create and destroy a pod. Fastest way to check basic cluster operation.",
	},
	E2eModeNonDisruptiveConformance: {
		name: E2eModeNonDisruptiveConformance, focus: conformanceFocus, skip: nonDisruptiveSkipList,
		desc: "Non-destructive conformance mode runs all of the conformance tests except those that would disrupt other cluster operations (e.g. tests that may cause nodes to be restarted or impact cluster permissions).",
	},
	E2eModeCertifiedConformance: {
		name: E2eModeCertifiedConformance, focus: conformanceFocus,
		desc: "Certified conformance mode runs the entire conformance suite, even disruptive tests. This is typically run in a dev environment to earn the CNCF Certified Kubernetes status.",
	},
	E2eModeConformanceLite: {
		name: E2eModeConformanceLite, focus: conformanceFocus, skip: genLiteSkips(), parallel: true,
		desc: "An unofficial mode of running the e2e tests which removes some of the longest running tests so that your tests can complete in the fastest time possible while maximizing coverage.",
	},
}

func genLiteSkips() string {
	quoted := make([]string, len(liteSkips))
	for i, v := range liteSkips {
		quoted[i] = regexp.QuoteMeta(v)
		// Quotes will cause the regexp to explode; easy to just change them to wildcards without an issue.
		quoted[i] = strings.ReplaceAll(quoted[i], `"`, ".")
	}
	return strings.Join(quoted, "|")
}

func validE2eModes() []string {
	keys := []string{}
	for key := range validModes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

type modesOptions struct {
	verbose bool
}

func NewCmdModes() *cobra.Command {
	f := modesOptions{}
	var modesCmd = &cobra.Command{
		Use:   "modes",
		Short: "Display the various modes in which to run the e2e plugin",
		Run: func(cmd *cobra.Command, args []string) {
			showModes(f)
		},
		Args: cobra.ExactArgs(0),
	}

	modesCmd.Flags().BoolVar(&f.verbose, "verbose", false, "Do not truncate output for each mode.")
	return modesCmd
}

func showModes(opt modesOptions) {
	count := 0
	if !opt.verbose {
		count = 200
	}
	for i, key := range validE2eModes() {
		opt := validModes[key]
		if i != 0 {
			fmt.Println("")
		}
		fmt.Println(truncate(fmt.Sprintf("Mode: %v", opt.name), count))
		fmt.Println(truncate(fmt.Sprintf("Description: %v", opt.desc), count))
		fmt.Println(truncate(fmt.Sprintf("E2E_FOCUS: %v", opt.focus), count))
		fmt.Println(truncate(fmt.Sprintf("E2E_SKIP: %v", opt.skip), count))
		fmt.Println(truncate(fmt.Sprintf("E2E_PARALLEL: %v", opt.parallel), count))
	}
}
func truncate(s string, count int) string {
	if count <= 0 {
		return s
	}
	if len(s) <= count {
		return s
	}
	return s[0:count] + "... (truncated) ..."
}
