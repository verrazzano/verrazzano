// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Defines timeouts for WebLogic Deploy Tool.
// +k8s:openapi-gen=true
type WDTTimeouts struct {
	// WDT activate WebLogic configuration changes timeout in milliseconds. Default: 180000.
	ActivateTimeoutMillis int `json:"activateTimeoutMillis,omitempty"`

	// WDT connect to WebLogic admin server timeout in milliseconds. Default: 120000.
	ConnectTimeoutMillis int `json:"connectTimeoutMillis,omitempty"`

	// WDT application or library deployment timeout in milliseconds. Default: 180000.
	DeployTimeoutMillis int `json:"deployTimeoutMillis,omitempty"`

	// WDT application or library redeployment timeout in milliseconds. Default: 180000.
	RedeployTimeoutMillis int `json:"redeployTimeoutMillis,omitempty"`

	// WDT set server groups timeout for extending a JRF domain configured cluster in milliseconds. Default: 180000.
	SetServerGroupsTimeoutMillis int `json:"setServerGroupsTimeoutMillis,omitempty"`

	// WDT application start timeout in milliseconds. Default: 180000.
	StartApplicationTimeoutMillis int `json:"startApplicationTimeoutMillis,omitempty"`

	// WDT application stop timeout in milliseconds. Default: 180000.
	StopApplicationTimeoutMillis int `json:"stopApplicationTimeoutMillis,omitempty"`

	// WDT application or library undeployment timeout in milliseconds. Default: 180000.
	UndeployTimeoutMillis int `json:"undeployTimeoutMillis,omitempty"`
}
