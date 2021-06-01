// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Defines online updates for Model In Image dynamic updates
// +k8s:openapi-gen=true
type OnlineUpdate struct {
	// Enable online update. Default is 'false'.
	Enabled bool `json:"enabled,omitempty"`

	// Controls behavior when non-dynamic WebLogic configuration changes are detected during an online update.
	// Non-dynamic changes are changes that require a domain restart to take effect. Valid values are 'CommitUpdateOnly'
	// (default), and 'CommitUpdateAndRoll'.
	OnNonDynamicChanges string `json:"onNonDynamicChanges,omitempty"`

	// Timeouts for WebLogic Deploy Tool.
	WDTTimeouts WDTTimeouts `json:"wdtTimeouts,omitempty"`
}
