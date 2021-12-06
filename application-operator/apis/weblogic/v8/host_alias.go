// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Host alias defines an IP and host names that map to the given IP
// +k8s:openapi-gen=true
type HostAlias struct {
	// WebLogic cluster name, if the server is part of a cluster
	IP string `json:"ip"`

	// Host names that map to the IP.
	// +x-kubernetes-list-type=set
	HostNames []string `json:"hostnames,omitempty"`
}
