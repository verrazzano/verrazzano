// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// KubernetesResource represents a generic Kubernetes resource
type KubernetesResource struct {
	// The annotations to be attached to generated resources.
	Annotations map[string]string `json:"annotations,omitempty"`

	// The labels to be attached to generated resources. The label names must
	// not start with 'weblogic.
	Labels map[string]string `json:"labels,omitempty"`
}
