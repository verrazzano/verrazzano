// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// +k8s:openapi-gen=true
type Model struct {
	// Name of a ConfigMap containing the WebLogic Deploy Tooling model.
	ConfigMap string `json:"configMap,omitempty"`

	// WebLogic Deploy Tooling domain type. Legal values: WLS, RestrictedJRF, JRF. Defaults to WLS.
	DomainType string `json:"domainType,omitempty"`

	// Runtime encryption secret. Required when domainHomeSourceType is set to FromModel.
	RuntimeEncryptionSecret string `json:"runtimeEncryptionSecret,omitempty"`
}
