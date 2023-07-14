// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

// chartValues Struct representing the verrazzano-network-policies Helm chart values
// Each field in this struct must have a corresponding key in the values.yaml file for the chart.
// This is only needed for the enabled/disable field for various components
// used by the network policies helm charts
type chartValues struct {
	Name                string                   `json:"name,omitempty"`
	ApplicationOperator *appOperatorValues       `json:"applicationOperator,omitempty"`
	AuthProxy           *authproxyValues         `json:"authproxy,omitempty"`
	CertManager         *certManagerValues       `json:"certManager,omitempty"`
	ConsoleValues       *consoleValues           `json:"console,omitempty"`
	ClusterOperator     *clusterOperatorValues   `json:"clusterOperator,omitempty"`
	CoherenceOperator   *coherenceOperatorValues `json:"coherenceOperator,omitempty"`
	WeblogicOperator    *weblogicOperatorValues  `json:"weblogicOperator,omitempty"`
	ElasticSearch       *elasticsearchValues     `json:"elasticSearch,omitempty"`
	Externaldns         *externalDNSValues       `json:"externaldns,omitempty"`
	Grafana             *grafanaValues           `json:"grafana,omitempty"`
	NGINX               *nginxValues             `json:"ingressNGINX,omitempty"`
	OAM                 *oamValues               `json:"oam,omitempty"`
	Istio               *istioValues             `json:"istio,omitempty"`
	JaegerOperator      *jaegerOperatorValues    `json:"jaegerOperator,omitempty"`
	Keycloak            *keycloakValues          `json:"keycloak,omitempty"`
	Prometheus          *prometheusValues        `json:"prometheus,omitempty"`
	Rancher             *rancherValues           `json:"rancher,omitempty"`
	Velero              *veleroValues            `json:"velero,omitempty"`
	ArgoCD              *argoCDValues            `json:"argoCd,omitempty"`
	FluentOperator      *fluentOperatorValues    `json:"fluentOperator,omitempty"`
	ClusterAPI          *clusterAPIValues        `json:"clusterAPI,omitempty"`
}

type authproxyValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type appOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type oamValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type certManagerValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type consoleValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type nginxValues struct {
	Enabled   bool   `json:"enabled"`
	Namespace string `json:"namespace"`
}

type elasticsearchValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type prometheusValues struct {
	Enabled bool `json:"enabled"` // Always write
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

type grafanaValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type externalDNSValues struct {
	Enabled   bool   `json:"enabled"`   // Always write
	Namespace string `json:"namespace"` // Always write
}

type jaegerOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type clusterOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type coherenceOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type weblogicOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type veleroValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type argoCDValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type clusterAPIValues struct {
	Enabled bool `json:"enabled"` // Always write
}

type fluentOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}
