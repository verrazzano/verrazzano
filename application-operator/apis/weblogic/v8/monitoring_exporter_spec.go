// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

// MonitoringExporterSpec defines the desired state of monitoring exporter sidecar
// +k8s:openapi-gen=true
type MonitoringExporterSpec struct {
	// The configuration for the WebLogic Monitoring Exporter. If WebLogic Server instances are already running and have
	// the monitoring exporter sidecar container, then changes to this field will be propagated to the exporter without
	// requiring the restart of the WebLogic Server instances.
	Configuration map[string]interface{} `json:"configuration,omitempty"`

	// The WebLogic Monitoring Exporter sidecar container image name.
	Image string `json:"image,omitempty"`

	// The image pull policy for the WebLogic Monitoring Exporter sidecar container image. Legal values are Always,
	// Never, and IfNotPresent. Defaults to Always if image ends in :latest; IfNotPresent, otherwise.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`
}
