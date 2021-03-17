// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// Configuration contains WebLogic Kubernetes Operator configuration options
// +k8s:openapi-gen=true
type Configuration struct {
	// The introspector job timeout value in seconds. If this field is specified, then the operator's ConfigMap
	// data.introspectorJobActiveDeadlineSeconds value is ignored. Defaults to 120 seconds.
	IntrospectorJobActiveDeadlineSeconds int `json:"introspectorJobActiveDeadlineSeconds,omitempty"`

	// Istio service mesh integration configuration
	Istio Istio `json:"istio,omitempty"`

	// Model in image model files and properties.
	Model Model `json:"model,omitempty"`

	// Settings for OPSS security.
	Opss Opss `json:"opss,omitempty"`

	// Determines how updated configuration overrides are distributed to already running WebLogic Server instances
	// following introspection when the domainHomeSourceType is PersistentVolume or Image. Configuration overrides
	// are generated during introspection from Secrets, the overrideConfigMap field, and WebLogic domain topology.
	// Legal values are DYNAMIC, which means that the operator will distribute updated configuration overrides
	// dynamically to running servers, and ON_RESTART, which means that servers will use updated configuration
	// overrides only after the server's next restart. The selection of ON_RESTART will not cause servers to restart
	// when there are updated configuration overrides available. See also domains.spec.introspectVersion.
	// Defaults to DYNAMIC.
	OverrideDistributionStrategy string `json:"overrideDistributionStrategy,omitempty"`

	// The name of the ConfigMap for WebLogic configuration overrides. If this field is specified, then the value
	// of spec.configOverrides is ignored.
	OverridesConfigMap string `json:"overridesConfigMap,omitempty"`

	// A list of names of the Secrets for WebLogic configuration overrides or model. If this field is specified, then
	// the value of spec.configOverrideSecrets is ignored.
	// +x-kubernetes-list-type=set
	Secrets []string `json:"secrets,omitempty"`
}
