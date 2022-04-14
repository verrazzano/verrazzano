// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

// vmiValues Struct representing the OpenSearch Helm chart values
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
type vmiValues struct {
	Config        *configValues        `json:"config,omitempty"`
	ElasticSearch *elasticsearchValues `json:"elasticSearch,omitempty"`
	Prometheus    *prometheusValues    `json:"prometheus,omitempty"`
	Grafana       *grafanaValues       `json:"grafana,omitempty"`
	Kibana        *kibanaValues        `json:"kibana,omitempty"`
}

type resourceRequestValues struct {
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage"` // Empty string allowed
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

type kibanaValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type grafanaValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type configValues struct {
	EnvName                 string `json:"envName,omitempty"`
	DNSSuffix               string `json:"dnsSuffix,omitempty"`
	EnableMonitoringStorage bool   `json:"enableMonitoringStorage,omitempty"`
}
