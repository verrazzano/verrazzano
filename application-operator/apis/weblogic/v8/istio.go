// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Istio service mesh integration configuration
// +k8s:openapi-gen=true
type Istio struct {
	// True, if this domain is deployed under an Istio service mesh.
	Enabled bool `json:"enabled,omitempty"`

	// The WebLogic readiness port for Istio. Defaults to 8888.
	ReadinessPort int `json:"readinessPort,omitempty"`

	// ReplicationChannelPort. The operator will create a `T3` protocol WebLogic network access point on each WebLogic
	// Server that is part of a cluster with this port to handle EJB and servlet session state replication traffic
	// between servers. This setting is ignored for clusters where the WebLogic cluster configuration already defines a
	// `replication-channel` attribute. Defaults to 4564.
	ReplicationChannelPort int `json:"replicationChannelPort,omitempty"`

	// LocalhostBindingsEnabled.  This setting was added in operator version 3.3.3, defaults to the Helm chart
	// configuration value `istioLocalhostBindingsEnabled` which in turn defaults to `true`. When `true`, the operator
	// creates a WebLogic network access point with a `localhost` binding for each existing channel and protocol.  Use
	// `true` for Istio versions prior to 1.10 and set to `false` for version 1.10 and later.
	LocalhostBindingsEnabled bool `json:"localhostBindingsEnabled,omitempty"`
}
