// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCAgentSecret contains information needed by the agent to access the admin cluster, such as the admin kubeconfig.
// This secret is used by the MC agent running on the managed cluster.
const MCAgentSecret = "verrazzano-cluster-agent" //nolint:gosec //#gosec G101

// MCRegistrationSecret - the name of the secret that contains the cluster registration information
const MCRegistrationSecret = "verrazzano-cluster-registration" //nolint:gosec //#gosec G101

// MCLocalRegistrationSecret - the name of the local secret that contains the cluster registration information.
// Thos is created at Verrazzano install.
const MCLocalRegistrationSecret = "verrazzano-local-registration" //nolint:gosec //#gosec G101

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

// CaBundleKey is the CA cert key in a secret
const CaBundleKey = "ca-bundle"

// CaFileDefault is the default value for the CA_FILE environment variable
const CaFileDefault = "/etc/ssl/certs/ca-bundle.crt"

// CaFileOverride is the location when the registration secret contains a ca-bundle
const CaFileOverride = "/fluentd/secret/ca-bundle"

// LabelVerrazzanoManaged - constant for a Kubernetes label that is applied by Verrazzano
const LabelVerrazzanoManaged = "verrazzano-managed"

// LabelVerrazzanoManagedDefault - default value for LabelVerrazzanoManaged
const LabelVerrazzanoManagedDefault = "true"

// LabelIstioInjection - constant for a Kubernetes label that is applied by Verrazzano
const LabelIstioInjection = "istio-injection"

// LabelVerrazzanoNamespace - constant for a Kubernetes label that is used by network policies
const LabelVerrazzanoNamespace = "verrazzano.io/namespace"

// LabelIstioInjectionDefault - default value for LabelIstioInjection
const LabelIstioInjectionDefault = "enabled"

// LabelWorkloadType - the type of workload, such as WebLogic
const LabelWorkloadType = "verrazzano.io/workload-type"

// WorkloadTypeWeblogic indicates the workload is WebLogic
const WorkloadTypeWeblogic = "weblogic"

// StatusUpdateChannelBufferSize - the number of status update messages that will be buffered
// by the agent channel before controllers trying to send more status updates will start blocking
const StatusUpdateChannelBufferSize = 50

// StatusUpdateBatchSize - the number of status update messages the multi cluster agent should
// process each time it wakes up
const StatusUpdateBatchSize = 10

// LabelUpgradeVersion - label which allows users to indicate that a running app should be upgraded to latest version of
// Verrazzano. When an application is deployed, the value of this label is set on workload.Status.CurrentUpgradeVersion.
// When reconciling, if the value provided in the label is different than the value in the workload status, the application
// will be 'upgraded' to use the resources provided by current version of Verrazzano. If any of these resources have
// changed since the application was deployed, the application will pick up the latest values and be restarted. If the
// label value matches the value in the workload status, all Verrazzano provided resources will remain unchanged.
const LabelUpgradeVersion = "upgrade-version"

// VzConsoleIngress - the name of the ingress for verrazzano console and api
const VzConsoleIngress = "verrazzano-ingress"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

// DefaultClusterName - the default cluster name
const DefaultClusterName = "local"

// VzPrometehusIngress - the name of the ingress for system vmi prometheus
const VzPrometheusIngress = "vmi-system-prometheus"
