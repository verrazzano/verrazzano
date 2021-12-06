// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import "time"

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoInstallNamespace is the namespace that the platform operator lives in
const VerrazzanoInstallNamespace = "verrazzano-install"

// Verrazzano is the name of the Verrazzano secret in the Verrazzano system namespace
const Verrazzano = "verrazzano"

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

// VzConsoleIngress - the name of the ingress for Verrazzano console and api
const VzConsoleIngress = "verrazzano-ingress"

// KeycloakIngress - the name of the ingress for Keycloak console and api
const KeycloakIngress = "keycloak"

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

// GlobalImagePullSecName is the name of the global image pull secret
const GlobalImagePullSecName = "verrazzano-container-registry"

// IngressNginxNamespace is the nginx ingress namespace name
const IngressNginxNamespace = "ingress-nginx"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

//  KeycloakNamespace is the keycloak namespace name
const KeycloakNamespace = "keycloak"

// VerrazzanoAuthProxyServiceName is the name of the Verrazzano auth proxy service
const VerrazzanoAuthProxyServiceName = "verrazzano-authproxy"

// VerrazzanoAuthProxyServicePort is the port exposed by the Verrazzano auth proxy service
const VerrazzanoAuthProxyServicePort = 8775

// VerrazzanoSystemTLSSecretName is the name of the system TLS secret
const VerrazzanoSystemTLSSecretName = "system-tls"

// The default name for install environment
const DefaultEnvironmentName = "default"

// Verrazzano version string for 1.0.0
const VerrazzanoVersion1_0_0 = "1.0.0"

// Verrazzano version string for 1.1.0
const VerrazzanoVersion1_1_0 = "1.1.0"

// VerrazzanoRestartAnnotation is the annotation used to restart platform workloads
const VerrazzanoRestartAnnotation = "verrazzano.io/restartedAt"

// UpgradeRetryVersion is the restart version annotation field
const UpgradeRetryVersion = "verrazzano.io/upgrade-retry-version"

// ObservedUpgradeRetryVersion is the previous restart version annotation field
const ObservedUpgradeRetryVersion = "verrazzano.io/observed-upgrade-retry-version"

// NGINXControllerServiceName
const NGINXControllerServiceName = "ingress-controller-ingress-nginx-controller"

// InstallOperation is the install string
const InstallOperation = "install"

// UpgradeOperation is the install string
const UpgradeOperation = "upgrade"

// InitializeOperation is the initialize string
const InitializeOperation = "initialize"

// ReconcileLoopRequeueInterval is the interval before reconcile gets called again.
const ReconcileLoopRequeueInterval = 3 * time.Minute
