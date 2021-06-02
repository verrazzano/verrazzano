// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// ClusterService represents a generic Kubernetes resource
type ClusterService struct {
	// The annotations to be attached to generated resources.
	Annotations map[string]string `json:"annotations,omitempty"`

	// The labels to be attached to generated resources. The label names must
	// not start with 'weblogic.
	Labels map[string]string `json:"labels,omitempty"`

	// Supports "ClientIP" and "None". Used to maintain session affinity. Enable client IP based session affinity.
	// Must be ClientIP or None. Defaults to None.
	// More info: https://kubernetes.io/docs/concepts/services-networking/service/#virtual-ips-and-service-proxies
	SessionAffinity string `json:"sessionAffinity,omitempty"`
}
