# Copyright 2017 Heptio Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

local k = import "ksonnet.beta.2/k.libsonnet";

local conf = {
    namespace: "heptio-sonobuoy",
    labels: {
        component: "sonobuoy",
    },
    serviceAccount: {
        name: "sonobuoy-serviceaccount",
    },

    # shared metadata
    metadata: {
        name: $.serviceAccount.name,
        labels: $.labels,
    },

};

local namespace =
    local ns = k.core.v1.namespace;
    ns.new() +
    ns.mixin.metadata.name(conf.namespace);

local serviceaccount =
    local sa = k.core.v1.serviceAccount;
    sa.new() +
    sa.mixin.metadata.mixinInstance(conf.metadata) +
    sa.mixin.metadata.namespace(conf.namespace);

local clusterRoleBinding =
    local crb = k.rbac.v1beta1.clusterRoleBinding;
    crb.new() +
    crb.mixin.metadata.mixinInstance(conf.metadata) +
    # TODO: replace with `crb.mixinroleRef.kind("ClusterRole") when https://github.com/ksonnet/ksonnet-lib/issues/53 closes.
    {roleRef: {kind: "ClusterRole"}} +
    crb.mixin.roleRef.apiGroup("rbac.authorization.k8s.io") +
    crb.mixin.roleRef.name(conf.serviceAccount.name) +
    crb.subjects([
        # TODO: replace with `crb.subjectsType.kind("ServiceAccount")` when https://github.com/ksonnet/ksonnet-lib/issues/43 closes.
        {kind: "ServiceAccount"} +
        crb.subjectsType.name(conf.serviceAccount.name) +
        crb.subjectsType.namespace(conf.namespace),
    ]);

local clusterRole =
    local cr = k.rbac.v1beta1.clusterRole;
    cr.new() +
    cr.mixin.metadata.mixinInstance(conf.metadata) +
    cr.mixin.metadata.namespace(conf.namespace) +
    cr.rules([
        cr.rulesType.apiGroups("*") +
        cr.rulesType.resources("*") +
        cr.rulesType.verbs("*"),
    ]);

local optRbacObj =
  if std.extVar("RBAC_ENABLED") != "0"
  then [clusterRoleBinding, clusterRole]
  else [];

k.core.v1.list.new([
  namespace,
  serviceaccount
] + optRbacObj)
