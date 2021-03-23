// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// ServerService represents a Kubernetes service for a WebLogic server
// +k8s:openapi-gen=true
type ServerService struct {
	// The annotations to be added to generated resources.
	Annotations map[string]string `json:"annotations,omitempty"`

	// The labels to be added to generated resources. The label names must not start with "weblogic.".
	Labels map[string]string `json:"labels,omitempty"`

	// If true, operator will create server services even for server instances without running pods.
	PrecreateService bool `json:"precreateService"`
}
