// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Model contains details of a WebLogic Deploy Tooling model
// +k8s:openapi-gen=true
type Model struct {
	// Name of a ConfigMap containing the WebLogic Deploy Tooling model.
	ConfigMap string `json:"configMap,omitempty"`

	// WebLogic Deploy Tooling domain type. Legal values: WLS, RestrictedJRF, JRF. Defaults to WLS.
	DomainType string `json:"domainType,omitempty"`

	// Location of the WebLogic Deploy Tooling model home. Defaults to /u01/wdt/models.
	ModelHome string `json:"modelHome,omitempty"`

	// Online update option for Model In Image dynamic update.
	OnlineUpdate OnlineUpdate `json:"onlineUpdate,omitempty"`

	// Runtime encryption secret. Required when domainHomeSourceType is set to FromModel.
	RuntimeEncryptionSecret string `json:"runtimeEncryptionSecret,omitempty"`

	// Location of the WebLogic Deploy Tooling installation. Defaults to /u01/wdt/weblogic-deploy.
	WDTInstallHome string `json:"wdtInstallHome,omitempty"`
}
