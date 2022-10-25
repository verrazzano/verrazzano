// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProfileType is the type of install profile.
type ProfileType string

const (
	// Dev identifies the development install profile
	Dev ProfileType = "dev"
	// Prod identifies the production install profile
	Prod ProfileType = "prod"
	// ManagedCluster identifies the production managed-cluster install profile
	ManagedCluster ProfileType = "managed-cluster"
)
const (
	// LoadBalancer is an ingress type of LoadBalancer.  This is the default value.
	LoadBalancer IngressType = "LoadBalancer"
	// NodePort is an ingress type of NodePort.
	NodePort IngressType = "NodePort"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=verrazzanos
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=vz;vzs
// +kubebuilder:printcolumn:name="Available",type="string",JSONPath=".status.available",description="Available/Enabled Verrazzano Components."
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[-1:].type",description="The current status of the install/uninstall."
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="The current version of the Verrazzano installation."
// +genclient

// Verrazzano specifies the Verrazzano API.
type Verrazzano struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VerrazzanoSpec   `json:"spec,omitempty"`
	Status VerrazzanoStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VerrazzanoList contains a list of Verrazzano resources.
type VerrazzanoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Verrazzano `json:"items"`
}

// VerrazzanoSpec defines the desired state of Verrazzano resource.
type VerrazzanoSpec struct {
	// The Verrazzano components.
	// +optional
	// +patchStrategy=merge
	Components ComponentSpec `json:"components,omitempty" patchStrategy:"merge"`
	// Defines the type of volume to be used for persistence for all components unless overridden, and can be one of
	// either EmptyDirVolumeSource or PersistentVolumeClaimVolumeSource. If PersistentVolumeClaimVolumeSource is
	// declared, then the claimName must reference the name of an existing VolumeClaimSpecTemplate declared in the
	// volumeClaimSpecTemplates section.
	// +optional
	// +patchStrategy=replace
	DefaultVolumeSource *corev1.VolumeSource `json:"defaultVolumeSource,omitempty" patchStrategy:"replace"`
	// Name of the installation. This name is part of the endpoint access URLs that are generated.
	// The default value is default.
	// +optional
	EnvironmentName string `json:"environmentName,omitempty"`
	// The installation profile to select. Valid values are prod (production), dev (development), and managed-cluster.
	// The default is prod.
	// +optional
	Profile ProfileType `json:"profile,omitempty"`
	// Security specifies Verrazzano security configuration.
	// +optional
	Security SecuritySpec `json:"security,omitempty"`
	// The version to install. Valid versions can be found
	// <a href="https://github.com/verrazzano/verrazzano/releases/">here</a>.
	// Defaults to the current version supported by the Verrazzano platform operator.
	// +optional
	Version string `json:"version,omitempty"`
	// Defines a named set of PVC configurations that can be referenced from components to configure persistent volumes.
	// +optional
	// +patchStrategy=merge,retainKeys
	VolumeClaimSpecTemplates []VolumeClaimSpecTemplate `json:"volumeClaimSpecTemplates,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
}

// SecuritySpec defines the security configuration for Verrazzano.
type SecuritySpec struct {
	// Specifies subjects that should be bound to the verrazzano-admin role.
	// +optional
	AdminSubjects []rbacv1.Subject `json:"adminSubjects,omitempty"`
	// Specifies subjects that should be bound to the verrazzano-monitor role.
	// +optional
	MonitorSubjects []rbacv1.Subject `json:"monitorSubjects,omitempty"`
}

// VolumeClaimSpecTemplate Contains common PVC configuration that can be referenced from Components; these
// do not actually result in generated PVCs, but can be used to provide common configuration to components that
// declare a PersistentVolumeClaimVolumeSource.
type VolumeClaimSpecTemplate struct {
	// Metadata about the PersistentVolumeClaimSpec template.
	// +kubebuilder:pruning:PreserveUnknownFields
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// A `PersistentVolumeClaimSpec` template that can be referenced by a Component to override its default storage
	// settings for a profile. At present, only a subset of the `resources.requests` object are honored depending on
	// the component.
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// InstanceInfo details of installed Verrazzano instance maintained in status field.
type InstanceInfo struct {
	// The Console URL for this Verrazzano installation.
	ConsoleURL *string `json:"consoleUrl,omitempty"`
	// The Grafana URL for this Verrazzano installation.
	GrafanaURL *string `json:"grafanaUrl,omitempty"`
	// The Jaeger UI URL for this Verrazzano installation.
	JaegerURL *string `json:"jaegerUrl,omitempty"`
	// The KeyCloak URL for this Verrazzano installation.
	KeyCloakURL *string `json:"keyCloakUrl,omitempty"`
	// The Kiali URL for this Verrazzano installation.
	KialiURL *string `json:"kialiUrl,omitempty"`
	// The OpenSearch Dashboards URL for this Verrazzano installation.
	OpenSearchDashboardsURL *string `json:"openSearchDashboardsUrl,omitempty"`
	// The OpenSearch URL for this Verrazzano installation.
	OpenSearchURL *string `json:"openSearchUrl,omitempty"`
	// The Prometheus URL for this Verrazzano installation.
	PrometheusURL *string `json:"prometheusUrl,omitempty"`
	// The Rancher URL for this Verrazzano installation.
	RancherURL *string `json:"rancherUrl,omitempty"`
}

// VerrazzanoStatus defines the observed state of a Verrazzano resource.
type VerrazzanoStatus struct {
	// The summary of Verrazzano component availability.
	Available *string `json:"available,omitempty"`
	// States of the individual installed components.
	Components ComponentStatusMap `json:"components,omitempty"`
	// The latest available observations of an object's current state.
	Conditions []Condition `json:"conditions,omitempty"`
	// State of the Verrazzano custom resource.
	State VzStateType `json:"state,omitempty"`
	// The Verrazzano instance info.
	VerrazzanoInstance *InstanceInfo `json:"instance,omitempty"`
	// The version of Verrazzano that is installed.
	Version string `json:"version,omitempty"`
}

// ComponentStatusMap is a map of components status details.
type ComponentStatusMap map[string]*ComponentStatusDetails

// ComponentStatusDetails defines the observed state of a component.
type ComponentStatusDetails struct {
	// If a component is available for use.
	Available *bool `json:"available,omitempty"`
	// Information about the current state of a component.
	Conditions []Condition `json:"conditions,omitempty"`
	// The generation of the last Verrazzano resource the Component was successfully reconciled against.
	LastReconciledGeneration int64 `json:"lastReconciledGeneration,omitempty"`
	// Name of the component.
	Name string `json:"name,omitempty"`
	// The generation of the Verrazzano resource the Component is currently being reconciled against.
	ReconcilingGeneration int64 `json:"reconcilingGeneration,omitempty"`
	// The version of Verrazzano that is installed.
	State CompStateType `json:"state,omitempty"`
	// The version of Verrazzano that is installed.
	Version string `json:"version,omitempty"`
}

// ConditionType identifies the condition of the install/uninstall/upgrade which can be checked with `kubectl wait`.
type ConditionType string

const (
	// CondPreInstall means an install about to start.
	CondPreInstall ConditionType = "PreInstall"

	// CondInstallStarted means an install is in progress.
	CondInstallStarted ConditionType = "InstallStarted"

	// CondInstallComplete means the install job has completed its execution successfully
	CondInstallComplete ConditionType = "InstallComplete"

	// CondInstallFailed means the install job has failed during execution.
	CondInstallFailed ConditionType = "InstallFailed"

	// CondUninstallStarted means an uninstall is in progress.
	CondUninstallStarted ConditionType = "UninstallStarted"

	// CondUninstallComplete means the uninstall job has completed its execution successfully
	CondUninstallComplete ConditionType = "UninstallComplete"

	// CondUninstallFailed means the uninstall job has failed during execution.
	CondUninstallFailed ConditionType = "UninstallFailed"

	// CondUpgradeStarted means that an upgrade has been started.
	CondUpgradeStarted ConditionType = "UpgradeStarted"

	// CondUpgradePaused means that an upgrade has been paused awaiting a VZ version update.
	CondUpgradePaused ConditionType = "UpgradePaused"

	// CondUpgradeFailed means the upgrade has failed during execution.
	CondUpgradeFailed ConditionType = "UpgradeFailed"

	// CondUpgradeComplete means the upgrade has completed successfully
	CondUpgradeComplete ConditionType = "UpgradeComplete"
)

// Condition describes current state of an installation.
type Condition struct {
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Type of condition.
	Type ConditionType `json:"type"`
}

// VzStateType identifies the state of a Verrazzano installation.
type VzStateType string

const (
	// VzStateUninstalling is the state when an uninstall is in progress
	VzStateUninstalling VzStateType = "Uninstalling"

	// VzStateUpgrading is the state when an upgrade is in progress
	VzStateUpgrading VzStateType = "Upgrading"

	// VzStatePaused is the state when an upgrade is paused due to version mismatch
	VzStatePaused VzStateType = "Paused"

	// VzStateReady is the state when a Verrazzano resource can perform an uninstall or upgrade
	VzStateReady VzStateType = "Ready"

	// VzStateFailed is the state when an install/uninstall/upgrade has failed
	VzStateFailed VzStateType = "Failed"

	// VzStateReconciling is the state when a resource is in progress reconciling
	VzStateReconciling VzStateType = "Reconciling"
)

// CompStateType identifies the state of a component
type CompStateType string

const (
	// CompStateDisabled is the state for when a component is not currently installed
	CompStateDisabled CompStateType = "Disabled"

	// CompStatePreInstalling is the state when an install is about to be started
	CompStatePreInstalling CompStateType = "PreInstalling"

	// CompStateInstalling is the state when an install is in progress
	CompStateInstalling CompStateType = "Installing"

	// CompStateUninstalling is the state when an uninstall is in progress
	CompStateUninstalling CompStateType = "Uninstalling"

	// CompStateUninstalled is the state when a component has been uninstalled
	CompStateUninstalled CompStateType = "Uninstalled"

	// CompStateUpgrading is the state when an upgrade is in progress
	CompStateUpgrading CompStateType = "Upgrading"

	// CompStateError is the state when a Verrazzano resource has experienced an error that may leave it in an unstable state
	CompStateError CompStateType = "Error"

	// CompStateReady is the state when a Verrazzano resource can perform an uninstall or upgrade
	CompStateReady CompStateType = "Ready"

	// CompStateFailed is the state when an install/uninstall/upgrade has failed
	CompStateFailed CompStateType = "Failed"
)

// ComponentSpec contains a set of components used by Verrazzano
type ComponentSpec struct {
	// The Application Operator component configuration.
	// +optional
	ApplicationOperator *ApplicationOperatorComponent `json:"applicationOperator,omitempty"`

	// The AuthProxy component configuration.
	// +optional
	AuthProxy *AuthProxyComponent `json:"authProxy,omitempty"`

	// The cert-manager component configuration.
	// +optional
	CertManager *CertManagerComponent `json:"certManager,omitempty"`

	// The Coherence Operator component configuration.
	// +optional
	CoherenceOperator *CoherenceOperatorComponent `json:"coherenceOperator,omitempty"`

	// The Verrazzano Console component configuration.
	// +optional
	Console *ConsoleComponent `json:"console,omitempty"`

	// The DNS component configuration.
	// +optional
	// +patchStrategy=replace
	DNS *DNSComponent `json:"dns,omitempty" patchStrategy:"replace"`

	// The Fluentd component configuration.
	// +optional
	Fluentd *FluentdComponent `json:"fluentd,omitempty"`

	// The Grafana component configuration.
	// +optional
	Grafana *GrafanaComponent `json:"grafana,omitempty"`

	// The ingress NGINX component configuration.
	// +optional
	IngressNGINX *IngressNginxComponent `json:"ingressNGINX,omitempty"`

	// The Istio component configuration.
	// +optional
	Istio *IstioComponent `json:"istio,omitempty"`

	// The Jaeger Operator component configuration.
	// +optional
	JaegerOperator *JaegerOperatorComponent `json:"jaegerOperator,omitempty"`

	// The Keycloak component configuration.
	// +optional
	Keycloak *KeycloakComponent `json:"keycloak,omitempty"`

	// The Kiali component configuration.
	// +optional
	Kiali *KialiComponent `json:"kiali,omitempty"`

	// The kube-state-metrics  component configuration.
	// +optional
	KubeStateMetrics *KubeStateMetricsComponent `json:"kubeStateMetrics,omitempty"`

	// The MySQL Operator component configuration.
	// +optional
	MySQLOperator *MySQLOperatorComponent `json:"mySQLOperator,omitempty"`

	// The OAM component configuration.
	// +optional
	OAM *OAMComponent `json:"oam,omitempty"`

	// The OpenSearch component configuration.
	// +optional
	OpenSearch *OpenSearchComponent `json:"opensearch,omitempty"`

	// The OpenSearch Dashboards component configuration.
	// +optional
	OpenSearchDashboards *OpenSearchDashboardsComponent `json:"opensearchDashboards,omitempty"`

	// The Prometheus component configuration.
	// +optional
	Prometheus *PrometheusComponent `json:"prometheus,omitempty"`

	// The Prometheus Adapter component configuration.
	// +optional
	PrometheusAdapter *PrometheusAdapterComponent `json:"prometheusAdapter,omitempty"`

	// The Prometheus Node Exporter component configuration.
	// +optional
	PrometheusNodeExporter *PrometheusNodeExporterComponent `json:"prometheusNodeExporter,omitempty"`

	// The Prometheus Operator component configuration.
	// +optional
	PrometheusOperator *PrometheusOperatorComponent `json:"prometheusOperator,omitempty"`

	// The Prometheus Pushgateway component configuration.
	// +optional
	PrometheusPushgateway *PrometheusPushgatewayComponent `json:"prometheusPushgateway,omitempty"`

	// The Rancher component configuration.
	// +optional
	Rancher *RancherComponent `json:"rancher,omitempty"`

	// The Rancher Backup component configuration.
	// +optional
	RancherBackup *RancherBackupComponent `json:"rancherBackup,omitempty"`

	// The Velero component configuration.
	// +optional
	Velero *VeleroComponent `json:"velero,omitempty"`

	// The Verrazzano component configuration.
	// +optional
	Verrazzano *VerrazzanoComponent `json:"verrazzano,omitempty"`

	// The WebLogic Kubernetes Operator component configuration.
	// +optional
	WebLogicOperator *WebLogicOperatorComponent `json:"weblogicOperator,omitempty"`
}

// OpenSearchComponent specifies the OpenSearch configuration.
type OpenSearchComponent struct {
	// If true, then OpenSearch will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// A list of OpenSearch node groups. For sample usage, see
	// <a href="../../../../../docs/customize/opensearch/">Customize OpenSearch</a>.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	Nodes []OpenSearchNode `json:"nodes,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name"`
	// A list of <a href="https://opensearch.org/docs/1.2/im-plugin/ism/index/">Index State Management</a> policies
	// to enable on OpenSearch.
	// +optional
	Policies []vmov1.IndexManagementPolicy `json:"policies,omitempty"`
}

// OpenSearchNode specifies a node group in the OpenSearch cluster.
type OpenSearchNode struct {
	// Name of the node group.
	Name string `json:"name,omitempty"`
	// Node group replica count.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`
	// Kubernetes container resources for nodes in the node group.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Role(s) that nodes in the group will assume. May be `master`, `data`, and/or `ingest`.
	Roles []vmov1.NodeRole `json:"roles,omitempty"`
	// Storage settings for the node group.
	// +optional
	Storage *OpenSearchNodeStorage `json:"storage,omitempty"`
}

type OpenSearchNodeStorage struct {
	// Node group storage size expressed as a
	// <a href="https://kubernetes.io/docs/reference/kubernetes-api/common-definitions/quantity/#Quantity">Quantity</a>
	Size string `json:"size"`
}

// OpenSearchDashboardsComponent specifies the OpenSearch Dashboards configuration.
type OpenSearchDashboardsComponent struct {
	// If true, then OpenSearch Dashboards will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// The number of pods to replicate. The default is 1.
	Replicas *int32 `json:"replicas,omitempty"`
}

// KubeStateMetricsComponent specifies the kube-state-metrics configuration.
type KubeStateMetricsComponent struct {
	// If true, then kube-state-metrics will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/prometheus-community/kube-state-metrics/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// DatabaseInfo specifies the database connection information for the Grafana DB instance.
type DatabaseInfo struct {
	// The host of the database.
	Host string `json:"host,omitempty"`
	// The name of the database.
	Name string `json:"name,omitempty"`
}

// GrafanaComponent specifies the Grafana configuration.
type GrafanaComponent struct {
	// The information to configure a connection to an external Grafana database.
	// +optional
	Database *DatabaseInfo `json:"database,omitempty"`
	// If true, then Grafana will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// The number of pods to replicate. The default is 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
}

// PrometheusComponent specifies the Prometheus configuration.
type PrometheusComponent struct {
	// If true, then Prometheus will be installed.
	// This is a legacy setting; the preferred way to configure Prometheus is using the
	// [PrometheusOperatorComponent](#install.verrazzano.io/v1beta1.PrometheusOperatorComponent).
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}

// PrometheusAdapterComponent specifies the Prometheus Adapter configuration.
type PrometheusAdapterComponent struct {
	// If true, then Prometheus Adaptor will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/prometheus-community/prometheus-adapter/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// PrometheusNodeExporterComponent specifies the Prometheus Node Exporter configuration.
type PrometheusNodeExporterComponent struct {
	// If true, then Prometheus Node Exporter will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/prometheus-community/prometheus-node-exporter/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// PrometheusOperatorComponent specifies the Prometheus Operator configuration.
type PrometheusOperatorComponent struct {
	// If true, then Prometheus Operator will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/prometheus-community/kube-prometheus-stack/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// PrometheusPushgatewayComponent specifies the Prometheus Pushgateway configuration.
type PrometheusPushgatewayComponent struct {
	// If true, then Prometheus Pushgateway will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/prometheus-community/prometheus-pushgateway/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// CertManagerComponent specifies the cert-manager configuration.
type CertManagerComponent struct {
	// The certificate configuration.
	// +optional
	// +patchStrategy=replace
	Certificate Certificate `json:"certificate,omitempty" patchStrategy:"replace"`
	// If true, then cert-manager will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/cert-manager/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// CoherenceOperatorComponent specifies the Coherence Operator configuration.
type CoherenceOperatorComponent struct {
	// If true, then Coherence Operator will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/coherence-operator/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// ApplicationOperatorComponent specifies the Application Operator configuration.
type ApplicationOperatorComponent struct {
	// If true, then Application Operator will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/helm_config/charts/verrazzano-application-operator/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// AuthProxyComponent specifies the AuthProxy configuration.
type AuthProxyComponent struct {
	// If true, then AuthProxy will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/helm_config/charts/verrazzano-authproxy/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// OAMComponent specifies the OAM configuration.
type OAMComponent struct {
	// If true, then OAM will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/oam-kubernetes-runtime/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// VerrazzanoComponent specifies the Verrazzano configuration.
type VerrazzanoComponent struct {
	// If true, then Verrazzano will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/helm_config/charts/verrazzano/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// KialiComponent specifies the Kiali configuration.
type KialiComponent struct {
	// If true, then Kiali will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/kiali-server/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// ConsoleComponent specifies the Verrazzano Console configuration.
type ConsoleComponent struct {
	// If true, then Verrazzano Console will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/helm_config/charts/verrazzano-console/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// DNSComponent specifies the DNS configuration.
type DNSComponent struct {
	// External DNS configuration.
	// +optional
	External *External `json:"external,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/external-dns/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
	// Oracle Cloud Infrastructure DNS configuration.
	// +optional
	OCI *OCI `json:"oci,omitempty"`
	// Wildcard DNS configuration. This is the default with a domain of nip.io.
	// +optional
	Wildcard *Wildcard `json:"wildcard,omitempty"`
}

// IngressNginxComponent specifies the ingress NGINX configuration.
type IngressNginxComponent struct {
	// If true, then ingress NGINX will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Name of the ingress class used by the ingress controller. Defaults to verrazzano-nginx.
	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/ingress-nginx/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
	// The list of port configurations used by the ingress.
	// +optional
	Ports []corev1.ServicePort `json:"ports,omitempty"`
	// The ingress type. Valid values are `LoadBalancer` and `NodePort`. The default value is LoadBalancer. If the ingress
	// type is NodePort, a valid and accessible IP address must be specified using the controller.service.externalIPs
	// key in the [InstallOverrides](#install.verrazzano.io/v1beta1.InstallOverrides). For sample usage, see
	// <a href="../../../../../docs/customize/externallbs/">External Load Balancers</a>.
	// +optional
	Type IngressType `json:"type,omitempty"`
}

// IstioComponent specifies the Istio configuration.
type IstioComponent struct {
	// If true, then ingress Istio will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// Istio sidecar injection enabled for installed components.  Default is true.
	// +optional
	InjectionEnabled *bool `json:"injectionEnabled,omitempty"`
	// List of Overrides for default IstioOperator. Overrides are merged together, but in the event of conflicting
	// fields, the last override in the list takes precedence over any others. You can find all possible values
	// <a href="https://istio.io/v1.13/docs/reference/config/istio.operator.v1alpha1/#IstioOperatorSpec">here</a>
	// Passing through an invalid IstioOperator resource will result in an error.
	// +optional
	InstallOverrides `json:",inline"`
}

// IsInjectionEnabled is istio sidecar injection enabled check.
func (c *IstioComponent) IsInjectionEnabled() bool {
	if c.Enabled == nil || *c.Enabled {
		return c.InjectionEnabled == nil || *c.InjectionEnabled
	}
	return c.InjectionEnabled != nil && *c.InjectionEnabled
}

// JaegerOperatorComponent specifies the Jaeger Operator configuration.
type JaegerOperatorComponent struct {
	// If true, then Jaeger Operator will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/jaegertracing/jaeger-operator/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// KeycloakComponent specifies the Keycloak configuration.
type KeycloakComponent struct {
	// If true, then Keycloak will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/keycloak/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
	// Contains the MySQL component configuration needed for Keycloak.
	// +optional
	MySQL MySQLComponent `json:"mysql,omitempty"`
}

// MySQLComponent specifies the MySQL configuration.
type MySQLComponent struct {
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/mysql/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
	// Defines the type of volume to be used for persistence for Keycloak/MySQL, and can be one of either
	// EmptyDirVolumeSource or PersistentVolumeClaimVolumeSource. If PersistentVolumeClaimVolumeSource is declared,
	// then the `claimName` must reference the name of a `VolumeClaimSpecTemplate` declared in the
	// `volumeClaimSpecTemplates` section.
	// +optional
	// +patchStrategy=replace
	VolumeSource *corev1.VolumeSource `json:"volumeSource,omitempty" patchStrategy:"replace"`
}

// MySQLOperatorComponent specifies the MySQL Operator configuration.
type MySQLOperatorComponent struct {
	// If true, then MySQL Operator will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/mysql-operator/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// RancherComponent specifies the Rancher configuration.
type RancherComponent struct {
	// If true, then Rancher will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/rancher/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
	// KeycloakAuthEnabled specifies whether the Keycloak Auth provider is enabled.  Default is false.
	// +optional
	KeycloakAuthEnabled *bool `json:"keycloakAuthEnabled,omitempty"`
}

// RancherBackupComponent specifies the Rancher Backup configuration.
type RancherBackupComponent struct {
	// If true, then Rancher Backup will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/rancher-backup/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// FluentdComponent specifies the Fluentd configuration.
type FluentdComponent struct {
	// If true, then Fluentd will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// A list of host path volume mounts, in addition to `/var/log`, into the Fluentd DaemonSet. The Fluentd component
	// collects log files in the /var/log/containers directory of Kubernetes worker nodes. The `/var/log/containers`
	// directory may contain symbolic links to files located outside the `/var/log` directory. If the host path
	// directory containing the log files is located outside `/var/log`, the Fluentd DaemonSet must have the volume
	// mount of that directory to collect the logs.
	// +optional
	// +patchStrategy=merge,retainKeys
	ExtraVolumeMounts []VolumeMount `json:"extraVolumeMounts,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"source"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/helm_config/charts/verrazzano-fluentd/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
	// The Oracle Cloud Infrastructure Logging configuration.
	// +optional
	OCI *OciLoggingConfiguration `json:"oci,omitempty"`
	// The secret containing the credentials for connecting to OpenSearch. This secret needs to be created in the
	// `verrazzano-install` namespace prior to creating the Verrazzano custom resource. Specify the OpenSearch login
	// credentials in the `username` and `password` fields in this secret. Specify the CA for verifying the OpenSearch
	// certificate in the `ca-bundle field`, if applicable. The default `verrazzano` is the secret for connecting to
	// the VMI OpenSearch.
	// +optional
	OpenSearchSecret string `json:"opensearchSecret,omitempty"`
	// The target OpenSearch URLs.
	// Specify this option in this <a href="https://docs.fluentd.org/output/elasticsearch#hosts-optional">format</a>.
	// The default `http://vmi-system-es-ingest-oidc:8775` is the VMI OpenSearch URL.
	// +optional
	OpenSearchURL string `json:"opensearchURL,omitempty"`
}

// WebLogicOperatorComponent specifies the WebLogic Kubernetes Operator configuration.
type WebLogicOperatorComponent struct {
	// If true, then WebLogic Kubernetes Operator will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/weblogic-operator/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// VeleroComponent  specifies the Velero configuration.
type VeleroComponent struct {
	// If true, then Velero will be installed.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
	// List of Overrides for the default `values.yaml` file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others. You can
	// find all possible values
	// [here]( {{% release_source_url path=platform-operator/thirdparty/charts/velero/values.yaml %}} )
	// and invalid values will be ignored.
	// +optional
	InstallOverrides `json:",inline"`
}

// VolumeMount defines a hostPath type Volume mount.
type VolumeMount struct {
	// The destination path on the Fluentd Container, defaults to the source host path.
	// +optional
	Destination string `json:"destination,omitempty"`
	// Specifies if the volume mount is read-only, defaults to true.
	// +optional
	ReadOnly *bool `json:"readOnly,omitempty"`
	// The source host path.
	Source string `json:"source"`
}

// ProviderType identifies Acme provider type.
type ProviderType string

const (
	// LetsEncrypt is a Let's Encrypt provider
	LetsEncrypt ProviderType = "LetsEncrypt"
)

// Acme identifies the ACME cert issuer.
type Acme struct {
	// Email address of the user.
	// +optional
	EmailAddress string `json:"emailAddress,omitempty"`
	// Environment.
	// +optional
	Environment string `json:"environment,omitempty"`
	// Name of the Acme provider.
	Provider ProviderType `json:"provider"`
}

// CA identifies the Certificate Authority cert issuer.
type CA struct {
	// The secret namespace.
	ClusterResourceNamespace string `json:"clusterResourceNamespace"`
	// The secret name.
	SecretName string `json:"secretName"`
}

// Certificate represents the type of cert issuer for an installation.
type Certificate struct {
	// The ACME configuration. Either acme or ca must be specified.
	// +optional
	Acme Acme `json:"acme,omitempty"`
	// The ACME configuration. Either acme or ca must be specified.
	// +optional
	CA CA `json:"ca,omitempty"`
}

// OciPrivateKeyFileName is the private key file name.
const OciPrivateKeyFileName = "oci_api_key.pem"

// OciConfigSecretFile is the name of the Oracle Cloud Infrastructure configuration yaml file.
const OciConfigSecretFile = "oci.yaml"

// Wildcard DNS type.
type Wildcard struct {
	// The type of wildcard DNS domain. For example, nip.io, sslip.io, and such.
	Domain string `json:"domain"`
}

// OCI DNS type.
type OCI struct {
	// Scope of the Oracle Cloud Infrastructure DNS zone (PRIVATE, GLOBAL). If not specified, then defaults to GLOBAL.
	// +optional
	DNSScope string `json:"dnsScope,omitempty"`
	// The Oracle Cloud Infrastructure DNS compartment OCID.
	DNSZoneCompartmentOCID string `json:"dnsZoneCompartmentOCID"`
	// The Oracle Cloud Infrastructure DNS zone OCID.
	DNSZoneOCID string `json:"dnsZoneOCID"`
	// Name of Oracle Cloud Infrastructure DNS zone.
	DNSZoneName string `json:"dnsZoneName"`
	// Name of the Oracle Cloud Infrastructure configuration secret. Generate a secret based on the
	// Oracle Cloud Infrastructure configuration profile you want to use. You can specify a profile other than
	// DEFAULT and specify the secret name. See instructions by running `./install/create_oci_config_secret.sh`.
	OCIConfigSecret string `json:"ociConfigSecret"`
}

// External DNS type.
type External struct {
	// The suffix for DNS names.
	Suffix string `json:"suffix"`
}

// IngressType is the type of ingress.
type IngressType string

func init() {
	SchemeBuilder.Register(&Verrazzano{}, &VerrazzanoList{})
}

// OciLoggingConfiguration is the Oracle Cloud Infrastructure logging configuration for Fluentd.
type OciLoggingConfiguration struct {
	// The OCID of the Oracle Cloud Infrastructure Log that will collect application logs.
	// +optional
	APISecret string `json:"apiSecret,omitempty"`
	// The OCID of the Oracle Cloud Infrastructure Log that will collect application logs.
	DefaultAppLogID string `json:"defaultAppLogId"`
	// The OCID of the Oracle Cloud Infrastructure Log that will collect system logs.
	SystemLogID string `json:"systemLogId"`
}

// InstallOverrides are used to pass installation overrides to components.
type InstallOverrides struct {
	// If false, then Verrazzano updates will ignore any configuration changes to this component. Defaults to `true`.
	// +optional
	MonitorChanges *bool `json:"monitorChanges,omitempty"`
	// List of overrides for the default values.yaml file for the component Helm chart. Overrides are merged together,
	// but in the event of conflicting fields, the last override in the list takes precedence over any others.
	// Invalid override values will be ignored.
	// +optional
	ValueOverrides []Overrides `json:"overrides,omitempty"`
}

// Overrides identifies overrides for a component.
type Overrides struct {
	// Selector for ConfigMap containing override data.
	// +optional
	ConfigMapRef *corev1.ConfigMapKeySelector `json:"configMapRef,omitempty"`
	// Selector for Secret containing override data.
	// +optional
	SecretRef *corev1.SecretKeySelector `json:"secretRef,omitempty"`
	// Configure overrides using inline YAML.
	// +optional
	Values *apiextensionsv1.JSON `json:"values,omitempty"`
}
