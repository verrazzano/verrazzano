// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

// verrazzanoValues Struct representing the Verrazzano Helm chart values
//
// In most cases, we only want to set overrides in this when they are present
// in the ComponentContext.EffectiveCR object, and use `omitempty` to prevent
// the JSON serialization code from writing out empty values.
//
// There are a few cases where this is not true
// - "enabled" flags should always be written; if the user or profile specifies false it
//   needs to be recorded in the overrides and not omitted
// - "resourceRequestValues.storage" should be allowed to record empty values, as it is a valid
//   value to the VMO to indicate ephemeral storage is to be used
//
type verrazzanoValues struct {
	Name                   string                        `json:"name,omitempty"`
	Global                 *globalValues                 `json:"global,omitempty"`
	Image                  *imageValues                  `json:"image,omitempty"`
	AppBinding             *appBindingValues             `json:"appBinding,omitempty"`
	ElasticSearch          *elasticsearchValues          `json:"elasticSearch,omitempty"`
	Prometheus             *prometheusValues             `json:"prometheus,omitempty"`
	Grafana                *grafanaValues                `json:"grafana,omitempty"`
	Kibana                 *kibanaValues                 `json:"kibana,omitempty"`
	Kiali                  *kialiValues                  `json:"kiali,omitempty"`
	Keycloak               *keycloakValues               `json:"keycloak,omitempty"`
	Rancher                *rancherValues                `json:"rancher,omitempty"`
	MonitoringOperator     *vmoValues                    `json:"monitoringOperator,omitempty"`
	NodeExporter           *nodeExporterValues           `json:"nodeExporter,omitempty"`
	Logging                *loggingValues                `json:"logging,omitempty"`
	Fluentd                *fluentdValues                `json:"fluentd,omitempty"`
	Console                *consoleValues                `json:"console,omitempty"`
	API                    *apiValues                    `json:"api,omitempty"`
	OCI                    *ociValues                    `json:"oci,omitempty"`
	Config                 *configValues                 `json:"config,omitempty"`
	Security               *securityRoleBindingValues    `json:"security,omitempty"`
	Kubernetes             *kubernetesValues             `json:"kubernetes,omitempty"`
	Externaldns            *externalDNSValues            `json:"externaldns,omitempty"`
	PrometheusOperator     *prometheusOperatorValues     `json:"prometheusOperator,omitempty"`
	PrometheusAdapter      *prometheusAdapterValues      `json:"prometheusAdapter,omitempty"`
	KubeStateMetrics       *kubeStateMetricsValues       `json:"kubeStateMetrics,omitempty"`
	PrometheusPushgateway  *prometheusPushgatewayValues  `json:"prometheusPushgateway,omitempty"`
	PrometheusNodeExporter *prometheusNodeExporterValues `json:"prometheusNodeExporter,omitempty"`
}

type subject struct {
	APIGroup  string `json:"apiGroup,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type volumeMount struct {
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	ReadOnly    bool   `json:"readOnly,omitempty"`
}

type resourceRequestValues struct {
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage"` // Empty string allowed
}

type imageValues struct {
	PullPolicy                    string `json:"pullPolicy,omitempty"`
	TerminationGracePeriodSeconds int    `json:"terminationGracePeriodSeconds"`
}

type globalValues struct {
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
}

type appBindingValues struct {
	UseSystemVMI bool `json:"useSystemVMI,omitempty"`
}

type elasticsearchValues struct {
	Enabled bool     `json:"enabled"` // Always write
	Nodes   *esNodes `json:"nodes,omitempty"`
}

type esNodes struct {
	Master *esNodeValues `json:"master,omitempty"`
	Data   *esNodeValues `json:"data,omitempty"`
	Ingest *esNodeValues `json:"ingest,omitempty"`
}

type esNodeValues struct {
	Replicas int                    `json:"replicas,omitempty"`
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type prometheusValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type kialiValues struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
}

type keycloakValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type rancherValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type kibanaValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type grafanaValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type vmoValues struct {
	Name                      string `json:"name,omitempty"`
	Enabled                   bool   `json:"enabled"` // Always write
	MetricsPort               int    `json:"metricsPort,omitempty"`
	DefaultSimpleCompReplicas int    `json:"defaultSimpleCompReplicas,omitempty"`
	DefaultPrometheusReplicas int    `json:"defaultPrometheusReplicas,omitempty"`
	AlertManagerImage         string `json:"alertManagerImage,omitempty"`
	EsWaitTargetVersion       string `json:"esWaitTargetVersion,omitempty"`
	OidcAuthEnabled           bool   `json:"oidcAuthEnabled,omitempty"`
	RequestMemory             string `json:"RequestMemory,omitempty"`
}

type nodeExporterValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type loggingValues struct {
	Name                string `json:"name,omitempty"`
	ElasticsearchURL    string `json:"elasticsearchURL,omitempty"`
	ElasticsearchSecret string `json:"elasticsearchSecret,omitempty"`
	ConfigHash          string `json:"configHash,omitempty"`
}

type fluentdValues struct {
	Enabled           bool                `json:"enabled"` // Always write
	ExtraVolumeMounts []volumeMount       `json:"extraVolumeMounts,omitempty"`
	OCI               *ociLoggingSettings `json:"oci,omitempty"`
}

type consoleValues struct {
	Enabled bool   `json:"enabled"` // Always write
	Name    string `json:"name,omitempty"`
}

type apiValues struct {
	Name string `json:"name,omitempty"`
	Port int    `json:"port,omitempty"`
}

type ociValues struct {
	Region      string               `json:"region,omitempty"`
	TenancyOcid string               `json:"tenancyOcid,omitempty"`
	UserOcid    string               `json:"userOcid,omitempty"`
	Fingerprint string               `json:"fingerprint,omitempty"`
	PrivateKey  string               `json:"privateKey,omitempty"`
	Compartment string               `json:"compartment,omitempty"`
	ClusterOcid string               `json:"clusterOcid,omitempty"`
	ObjectStore *objectStoreSettings `json:"objectStore,omitempty"`
}

type objectStoreSettings struct {
	BucketName string `json:"bucketName,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
}

type configValues struct {
	EnvName                 string `json:"envName,omitempty"`
	DNSSuffix               string `json:"dnsSuffix,omitempty"`
	EnableMonitoringStorage bool   `json:"enableMonitoringStorage,omitempty"`
}

type externalDNSValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type securityRoleBindingValues struct {
	AdminSubjects   map[string]subject `json:"adminSubjects,omitempty"`
	MonitorSubjects map[string]subject `json:"monitorSubjects,omitempty"`
}

type kubernetesValues struct {
	Service *serviceSettings `json:"service,omitempty"`
}

type serviceSettings struct {
	Endpoint *endpoint `json:"endpoint,omitempty"`
}

type endpoint struct {
	IP   string `json:"ip,omitempty"`
	Port int    `json:"port,omitempty"`
}

type ociLoggingSettings struct {
	DefaultAppLogID string `json:"defaultAppLogId"`
	SystemLogID     string `json:"systemLogId"`
	APISecret       string `json:"apiSecret,omitempty"`
}

type prometheusOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type prometheusAdapterValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type kubeStateMetricsValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type prometheusPushgatewayValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type prometheusNodeExporterValues struct {
	Enabled bool `json:"enabled"` // Always write
}
