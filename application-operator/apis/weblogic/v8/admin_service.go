// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// +k8s:openapi-gen=true
type AdminService struct {
	// The annotations to be added to generated resources.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Specifies which of the admin server's WebLogic channels should be exposed outside
	// the Kubernetes cluster via a node port service, along with the node port for
	// each channel. If not specified, the admin server's node port service will
	// not be created.
	// +x-kubernetes-list-type=set
	Channels []Channel `json:"channels,omitempty"`

	// The labels to be added to generated resources. The label names must not start with "weblogic.".
	Labels map[string]string `json:"labels,omitempty"`
}
