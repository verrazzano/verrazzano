// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// SystemTLS is the name of the system-tls secret in the Verrazzano system namespace
const SystemTLS = "system-tls"

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoInstallNamespace is the namespace that the platform operator lives in
const VerrazzanoInstallNamespace = "verrazzano-install"

// Verrazzano is the name of the verrazzano secret in the Verrazzano system namespace
const Verrazzano = "verrazzano"

// VerrazzanoPromInternal is the name of the Verrazzano internal Prometheus secret in the Verrazzano system namespace
const VerrazzanoPromInternal = "verrazzano-prom-internal"

// VerrazzanoESInternal is the name of the Verrazzano internal Elasticsearch secret in the Verrazzano system namespace
const VerrazzanoESInternal = "verrazzano-es-internal"

// VerrazzanoMultiClusterNamespace is the multi-cluster namespace for verrazzano
const VerrazzanoMultiClusterNamespace = "verrazzano-mc"

// MCAgentSecret contains information needed by the agent to access the admin cluster, such as the admin kubeconfig.
// This secret is used by the MC agent running on the managed cluster.
const MCAgentSecret = "verrazzano-cluster-agent" //nolint:gosec //#gosec G101

// MCRegistrationSecret contains information which related to the managed cluster itself, such as the
// managed cluster name.
const MCRegistrationSecret = "verrazzano-cluster-registration" //nolint:gosec //#gosec G101

// MCLocalRegistrationSecret - the name of the local secret that contains the cluster registration information.
// This is created at Verrazzano install.
const MCLocalRegistrationSecret = "verrazzano-local-registration" //nolint:gosec //#gosec G101

// MCClusterRole is the role name for the role used during VMC reconcile
const MCClusterRole = "verrazzano-managed-cluster"

// MCLocalCluster is the name of the local cluster
const MCLocalCluster = "local"

// AdminClusterConfigMapName is the name of the configmap that contains admin cluster server address
const AdminClusterConfigMapName = "verrazzano-admin-cluster"

// ServerDataKey is the key into ConfigMap data for cluster server address
const ServerDataKey = "server"

// VzConsoleIngress - the name of the ingress for verrazzano console and api
const VzConsoleIngress = "verrazzano-ingress"

// RegistryOverrideEnvVar is the environment variable name used to override the registry housing images we install
const RegistryOverrideEnvVar = "REGISTRY"

// ImageRepoOverrideEnvVar is the environment variable name used to set the image repository
const ImageRepoOverrideEnvVar = "IMAGE_REPO"

// VerrazzanoAppOperatorImageEnvVar is the environment variable used to override the Verrazzano Application Operator image
const VerrazzanoAppOperatorImageEnvVar = "APP_OPERATOR_IMAGE"

// The Kubernetes default namespace
const DefaultNamespace = "default"

const BomVerrazzanoVersion = "VERRAZZANO_VERSION"

// ClusterNameData - the field name in MCRegistrationSecret that contains this managed cluster's name
const ClusterNameData = "managed-cluster-name"

// ElasticsearchURLData - the field name in MCRegistrationSecret that contains the admin cluster's
// Elasticsearch endpoint's URL
const ElasticsearchURLData = "es-url"

// ClusterNameEnvVar is the environment variable used to identify the managed cluster for fluentd
const ClusterNameEnvVar = "CLUSTER_NAME"

// ElasticsearchURLEnvVar is the environment variable used to identify the admin clusters Elasticsearch URL
const ElasticsearchURLEnvVar = "ELASTICSEARCH_URL"
