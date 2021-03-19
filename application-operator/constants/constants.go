// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// VerrazzanoClustersGroup - clusters group
const VerrazzanoClustersGroup = "clusters.verrazzano.io"

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCAgentSecret contains information needed by the agent to access the admin cluster, such as the admin kubeconfig.
// This secret is used by the MC agent running on the managed cluster.
const MCAgentSecret = "verrazzano-cluster-agent"

// MCRegistrationSecret - the name of the secret that contains the cluster registration information
const MCRegistrationSecret = "verrazzano-cluster-registration"

// MCLocalRegistrationSecret - the name of the local secret that contains the cluster registration information.
// Thos is created at Verrazzano install.
const MCLocalRegistrationSecret = "verrazzano-local-registration"

// AdminKubeconfigData - the field name in MCRegistrationSecret that contains the admin cluster's kubeconfig
const AdminKubeconfigData = "admin-kubeconfig"

// ClusterNameData - the field name in MCRegistrationSecret that contains this managed cluster's name
const ClusterNameData = "managed-cluster-name"

// ElasticsearchURLData - the field name in MCRegistrationSecret that contains the admin cluster's
// Elasticsearch endpoint's URL
const ElasticsearchURLData = "es-url"

// ElasticsearchUsernameData - the field name in MCRegistrationSecret that contains the admin
// cluster's Elasticsearch username
const ElasticsearchUsernameData = "username"

// ElasticsearchPasswordData - the field name in MCRegistrationSecret that contains the admin
// cluster's Elasticsearch password
const ElasticsearchPasswordData = "password"

// ElasticsearchCABundleData - the field name in MCRegistrationSecret that contains the admin
// cluster's Elasticsearch CA bundle
const ElasticsearchCABundleData = "ca-bundle"

// LabelVerrazzanoManaged - constant for a Kubernetes label that is applied by Verrazzano
const LabelVerrazzanoManaged = "verrazzano-managed"

// LabelVerrazzanoManagedDefault - default value for LabelVerrazzanoManaged
const LabelVerrazzanoManagedDefault = "true"

// LabelIstioInjection - constant for a Kubernetes label that is applied by Verrazzano
const LabelIstioInjection = "istio-injection"

// LabelIstioInjectionDefault - default value for LabelIstioInjection
const LabelIstioInjectionDefault = "enabled"

// StatusUpdateChannelBufferSize - the number of status update messages that will be buffered
// by the agent channel before controllers trying to send more status updates will start blocking
const StatusUpdateChannelBufferSize = 50

// StatusUpdateBatchSize - the number of status update messages the multi cluster agent should
// process each time it wakes up
const StatusUpdateBatchSize = 10
