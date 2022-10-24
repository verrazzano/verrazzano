// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import "time"

// VerrazzanoSystemNamespace is the system namespace for verrazzano
const VerrazzanoSystemNamespace = "verrazzano-system"

// VerrazzanoInstallNamespace is the namespace that the platform operator lives in
const VerrazzanoInstallNamespace = "verrazzano-install"

// VerrazzanoMonitoringNamespace is the namespace for monitoring components
const VerrazzanoMonitoringNamespace = "verrazzano-monitoring"

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

// ElasticsearchIngress is the name of the ingress for Elasticsearch
const ElasticsearchIngress = "vmi-system-es-ingest"

// GrafanaIngress is the name of the ingress for Grafana
const GrafanaIngress = "vmi-system-grafana"

// KibanaIngress is the name of the ingress for Kibana
const KibanaIngress = "vmi-system-kibana"

// PrometheusIngress is the name of the ingress for Prometheus
const PrometheusIngress = "vmi-system-prometheus"

// GlobalImagePullSecName is the name of the global image pull secret
const GlobalImagePullSecName = "verrazzano-container-registry"

// IngressNginxNamespace is the nginx ingress namespace name
const IngressNginxNamespace = "ingress-nginx"

// IstioSystemNamespace - the Istio system namespace
const IstioSystemNamespace = "istio-system"

// RancherIngress is the name of the ingress for Rancher
const RancherIngress = "rancher"

// KialiIngress is the name of the ingress for Kiali
const KialiIngress = "vmi-system-kiali"

// JaegerIngress is the name of the ingress for Jaeger
const JaegerIngress = "verrazzano-jaeger"

// KeycloakNamespace is the keycloak namespace name
const KeycloakNamespace = "keycloak"

// KeycloakIngress - the name of the ingress for Keycloak console and api
const KeycloakIngress = "keycloak"

// ArgoCDNamespace - the name of the ArgoCD namespace
const ArgoCDNamespace = "argocd"

// ArgoCDIngress - the name of the ingress for Argo CD
const ArgoCDIngress = "argocd-server"

// VerrazzanoAuthProxyServiceName is the name of the Verrazzano auth proxy service
const VerrazzanoAuthProxyServiceName = "verrazzano-authproxy"

// VerrazzanoAuthProxyServicePort is the port exposed by the Verrazzano auth proxy service
const VerrazzanoAuthProxyServicePort = 8775

// DefaultEnvironmentName is the default name for install environment
const DefaultEnvironmentName = "default"

// VerrazzanoVersion1_0_0 is the Verrazzano version string for 1.0.0
const VerrazzanoVersion1_0_0 = "1.0.0"

// VerrazzanoVersion1_1_0 is the Verrazzano version string for 1.1.0
const VerrazzanoVersion1_1_0 = "1.1.0"

// VerrazzanoVersion1_3_0 is the Verrazzano version string for 1.3.0
const VerrazzanoVersion1_3_0 = "1.3.0"

// VerrazzanoVersion1_4_0 is the Verrazzano version string for 1.4.0
const VerrazzanoVersion1_4_0 = "1.4.0"

// UpgradeRetryVersion is the restart version annotation field
const UpgradeRetryVersion = "verrazzano.io/upgrade-retry-version"

// ObservedUpgradeRetryVersion is the previous restart version annotation field
const ObservedUpgradeRetryVersion = "verrazzano.io/observed-upgrade-retry-version"

// NGINXControllerServiceName is the nginx ingress controller name
const NGINXControllerServiceName = "ingress-controller-ingress-nginx-controller"

// InstallOperation indicates that an install operation being executed by a component
const InstallOperation = "install"

// UpgradeOperation indicates that an upgrade operation being executed by a component
const UpgradeOperation = "upgrade"

// UninstallOperation indicates that an uninstall operation being executed by a component
const UninstallOperation = "uninstall"

// InitializeOperation indicates that an initialize operation being executed by a component
const InitializeOperation = "initialize"

// ReconcileLoopRequeueInterval is the interval before reconcile gets called again.
const ReconcileLoopRequeueInterval = 3 * time.Minute

// VMISecret is the secret used for VMI
const VMISecret = "verrazzano"

// GrafanaSecret is the secret used for VMI
const GrafanaSecret = "grafana-admin"

// GrafanaDBSecret is the secret used for VMI
const GrafanaDBSecret = "grafana-db"

// VMIBackupSecretName is the backup VMI secret
const VMIBackupSecretName = "verrazzano-backup" //nolint:gosec //#gosec G101

// ObjectStoreAccessKey is used for the VMI backup secret
const ObjectStoreAccessKey = "object_store_access_key"

// ObjectStoreAccessSecretKey is used for the VMI backup secret
const ObjectStoreAccessSecretKey = "object_store_secret_key"

// VerrazzanoIngressSecret is the secret where the verrazzano/console TLS cert, key, and CA(s) are stored
const VerrazzanoIngressSecret = "verrazzano-tls" //nolint:gosec //#gosec G101

// VerrazzanoLocalCABundleSecret is a secret containing the admin ca bundle
const VerrazzanoLocalCABundleSecret = "verrazzano-local-ca-bundle" //nolint:gosec //#gosec G101

// KubernetesAppLabel is a label key for kubernetes apps
const KubernetesAppLabel = "app.kubernetes.io/component"

// JaegerCollectorService is a label value for Jaeger collector
const JaegerCollectorService = "service-collector"

// OverridesFinalizer is a label value for value override object finalizer
const OverridesFinalizer = "overrides.finalizers.verrazzano.io/finalizer"

// ConfigMapKind is a label value for ConfigMap kind
const ConfigMapKind = "ConfigMap"

// SecretKind is a label value for Secret Kind
const SecretKind = "Secret"

// PromManagedClusterCACertsSecretName is the name of the secret that contains managed cluster CA certificates. The secret is mounted
// as a volume in the Prometheus pod.
const PromManagedClusterCACertsSecretName = "managed-cluster-ca-certs"

// VerrazzanoComponentLabelKey is the key for the verrazzano component label to distinguish verrazzano component resources
const VerrazzanoComponentLabelKey = "verrazzano-component"

// IstioAppLabel is the label used for Verrazzano Istio components
const IstioAppLabel = "verrazzano.io/istio"

// OldReclaimPolicyLabel is the name of the label used to store the old reclaim policy of a persistent volume before it is migrated
const OldReclaimPolicyLabel = "verrazzano.io/old-reclaim-policy"

// StorageForLabel is the name of the label applied to a persistent volume so storage can be selected by a pod
const StorageForLabel = "verrazzano.io/storage-for"

// PrometheusStorageLabelValue is the label value for Prometheus storage
const PrometheusStorageLabelValue = "prometheus"

// VMISystemPrometheusVolumeClaim is the name of the VMO-managed Prometheus persistent volume claim
const VMISystemPrometheusVolumeClaim = "vmi-system-prometheus"

// VeleroNameSpace indicates the namespace to be used for velero installation
const VeleroNameSpace = "verrazzano-backup"

// ResticDaemonSetName indicates name of the daemonset that is installed as part of component velero
const ResticDaemonSetName = "restic"

// RancherBackupNamesSpace indicates the namespace to be used for Rancher Backup installation
const RancherBackupNamesSpace = "cattle-resources-system"
