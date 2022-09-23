// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

// chartValues Struct representing the verrazzano-network-policies Helm chart values
// Each field in this struct must have a corresponding key in the values.yaml file for the chart.
// This is only needed for the enabled/disable field for various components
// used by the network policies helm charts
type chartValues struct {
	Name           string                `json:"name,omitempty"`
	ElasticSearch  *elasticsearchValues  `json:"elasticSearch,omitempty"`
	Externaldns    *externalDNSValues    `json:"externaldns,omitempty"`
	Grafana        *grafanaValues        `json:"grafana,omitempty"`
	Istio          *istioValues          `json:"istio,omitempty"`
	JaegerOperator *jaegerOperatorValues `json:"jaegerOperator,omitempty"`
	Keycloak       *keycloakValues       `json:"keycloak,omitempty"`
	Prometheus     *prometheusValues     `json:"prometheus,omitempty"`
	Rancher        *rancherValues        `json:"rancher,omitempty"`
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
	Enabled bool `json:"enabled"` // Always write
}

type jaegerOperatorValues struct {
	Enabled bool `json:"enabled"` // Always write
}
