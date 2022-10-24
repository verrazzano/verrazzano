// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// VerrazzanoSystemNamespace is the system namespace for Verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for Verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCAgentSecret contains information needed by the agent to access the admin cluster, such as the admin kubeconfig.
// This secret is used by the MC agent running on the managed cluster.
const MCAgentSecret = "verrazzano-cluster-agent" //nolint:gosec //#gosec G101

// MCRegistrationSecret - the name of the secret that contains the cluster registration information
const MCRegistrationSecret = "verrazzano-cluster-registration" //nolint:gosec //#gosec G101

// MCLocalRegistrationSecret - the name of the local secret that contains the cluster registration information.
// Thos is created at Verrazzano install.
const MCLocalRegistrationSecret = "verrazzano-local-registration" //nolint:gosec //#gosec G101

// ClusterNameData - the field name in MCRegistrationSecret that contains this managed cluster's name
const ClusterNameData = "managed-cluster-name"

// OpensearchURLData - the field name in MCRegistrationSecret that contains the admin cluster's
// Elasticsearch endpoint's URL
const OpensearchURLData = "es-url"

// OpensearchUsernameData - the field name in MCRegistrationSecret that contains the admin
// cluster's Elasticsearch username
const OpensearchUsernameData = "username"

// OpensearchPasswordData - the field name in MCRegistrationSecret that contains the admin
// cluster's Elasticsearch password
const OpensearchPasswordData = "password"

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

// VzConsoleIngress - the name of the ingress for Verrazzano console and api
const VzConsoleIngress = "verrazzano-ingress"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

// DefaultClusterName - the default cluster name
const DefaultClusterName = "local"

// VzPrometehusIngress - the name of the ingress for system vmi prometheus
const VzPrometheusIngress = "vmi-system-prometheus"

// ClusterNameEnvVar is the environment variable used to identify the managed cluster for fluentd
const FluentdClusterNameEnvVar = "CLUSTER_NAME"

// FluentdOpensearchURLEnvVar is the environment variable name used to identify the admin cluster's
// Elasticsearch URL for fluentd
const FluentdOpensearchURLEnvVar = "ELASTICSEARCH_URL"

// FluentdOpensearchUserEnvVar is the environment variable name used to identify the admin cluster's
// Elasticsearch username for fluentd
const FluentdOpensearchUserEnvVar = "ELASTICSEARCH_USER"

// FluentdOpensearchPwdEnvVar is the environment variable name used to identify the admin cluster's
// Elasticsearch password for fluentd
const FluentdOpensearchPwdEnvVar = "ELASTICSEARCH_PASSWORD"

// VerrazzanoUsernameData - the field name in Verrazzano secret that contains the username
const VerrazzanoUsernameData = "username"

// VerrazzanoPasswordData - the field name in Verrazzano secret that contains the password
const VerrazzanoPasswordData = "password"

// MetricsWorkloadLabel - the label for identifying a pods scrape target
const MetricsWorkloadLabel = "app.verrazzano.io/workload"

// Webhook success status
const StatusReasonSuccess = "success"

// OCILoggingIDAnnotation Annotation name for a customized OCI log ID for all containers in a namespace
const OCILoggingIDAnnotation = "verrazzano.io/oci-log-id"

// WorkloadTypeCoherence indicates the workload is Coherence
const WorkloadTypeCoherence = "coherence"

// WorkloadTypeGeneric indicates the workload is generic (one of VerrazzanoHelidonWorkload, ContainerizedWorkload or Deployment)
const WorkloadTypeGeneric = "generic"

// VerrazzanoIngressTLSSecret is the name of the secret in a cluster that contains the cluster's ca bundle
const VerrazzanoIngressTLSSecret = "verrazzano-tls" //nolint:gosec //#gosec G101

// VerrazzanoLocalCABundleSecret is the name of the secret in the verrazzano-mc namespace on an admin cluster that contains the cluster's ca bundle
const VerrazzanoLocalCABundleSecret = "verrazzano-local-ca-bundle" //nolint:gosec //#gosec G101

// LegacyDefaultMetricsTemplateName is the name of the default metrics template used for standard
// Kubernetes workloads (legacy as of VZ 1.4)
const LegacyDefaultMetricsTemplateName = "standard-k8s-metrics-template"

// LegacyDefaultMetricsTemplateNamespace is the namespace containing the default metrics template
// used for standard Kubernetes workloads (legacy as of VZ 1.4)
const LegacyDefaultMetricsTemplateNamespace = VerrazzanoSystemNamespace

// DefaultScraperName is the default Prometheus deployment name used to scrape metrics. If a metrics trait does not specify a scraper, this
// is the scraper that will be used.
const DefaultScraperName = "verrazzano-system/vmi-system-prometheus-0"
