// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
)

// verrazzanoValues Struct representing the Verrazzano Helm chart values
//
// In most cases, we only want to set overrides in this when they are present
// in the ComponentContext.EffectiveCR object, and use `omitempty` to prevent
// the JSON serialization code from writing out empty values.
//
// There are a few cases where this is not true
//   - "enabled" flags should always be written; if the user or profile specifies false it
//     needs to be recorded in the overrides and not omitted
//   - "vmi.ResourceRequestValues.storage" should be allowed to record empty values, as it is a valid
//     value to the VMO to indicate ephemeral storage is to be used
type verrazzanoValues struct {
	Name                   string                        `json:"name,omitempty"`
	Global                 *globalValues                 `json:"global,omitempty"`
	Image                  *imageValues                  `json:"image,omitempty"`
	AppBinding             *appBindingValues             `json:"appBinding,omitempty"`
	ElasticSearch          *elasticsearchValues          `json:"elasticSearch,omitempty"`
	Prometheus             *prometheusValues             `json:"prometheus,omitempty"`
	Grafana                *grafanaValues                `json:"grafana,omitempty"`
	Kibana                 *kibanaValues                 `json:"kibana,omitempty"`
	Istio                  *istioValues                  `json:"istio,omitempty"`
	Kiali                  *kialiValues                  `json:"kiali,omitempty"`
	Keycloak               *keycloakValues               `json:"keycloak,omitempty"`
	Rancher                *rancherValues                `json:"rancher,omitempty"`
	API                    *apiValues                    `json:"api,omitempty"`
	Config                 *configValues                 `json:"config,omitempty"`
	Security               *securityRoleBindingValues    `json:"security,omitempty"`
	Kubernetes             *kubernetesValues             `json:"kubernetes,omitempty"`
	Externaldns            *externalDNSValues            `json:"externaldns,omitempty"`
	PrometheusOperator     *prometheusOperatorValues     `json:"prometheusOperator,omitempty"`
	PrometheusAdapter      *prometheusAdapterValues      `json:"prometheusAdapter,omitempty"`
	KubeStateMetrics       *kubeStateMetricsValues       `json:"kubeStateMetrics,omitempty"`
	PrometheusPushgateway  *prometheusPushgatewayValues  `json:"prometheusPushgateway,omitempty"`
	PrometheusNodeExporter *prometheusNodeExporterValues `json:"prometheusNodeExporter,omitempty"`
	JaegerOperator         *jaegerOperatorValues         `json:"jaegerOperator,omitempty"`
}

type subject struct {
	APIGroup  string `json:"apiGroup,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
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
	Enabled          bool `json:"enabled"` // Always write
	MultiNodeCluster bool `json:"multiNodeCluster"`
}

type prometheusValues struct {
	Enabled  bool                          `json:"enabled"` // Always write
	Requests *common.ResourceRequestValues `json:"requests,omitempty"`
}

type kialiValues struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
}

type keycloakValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type istioValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type rancherValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type kibanaValues struct {
	Enabled  bool                          `json:"enabled"` // Always write
	Requests *common.ResourceRequestValues `json:"requests,omitempty"`
}

type grafanaValues struct {
	Enabled  bool                          `json:"enabled"` // Always write
	Requests *common.ResourceRequestValues `json:"requests,omitempty"`
}

type apiValues struct {
	Name string `json:"name,omitempty"`
	Port int    `json:"port,omitempty"`
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

type jaegerOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}
