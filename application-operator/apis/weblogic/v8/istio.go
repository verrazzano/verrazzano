// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Istio service mesh integration configuration
// +k8s:openapi-gen=true
type Istio struct {
	// True, if this domain is deployed under an Istio service mesh.
	Enabled bool `json:"enabled,omitempty"`

	// The WebLogic readiness port for Istio. Defaults to 8888.
	ReadinessPort int `json:"readinessPort,omitempty"`
}
