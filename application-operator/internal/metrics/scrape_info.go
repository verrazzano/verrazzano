// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metrics

import corev1 "k8s.io/api/core/v1"

const (
	prometheusClusterNameLabel = "verrazzano_cluster"
)

// ScrapeInfo captures the information needed to construct the service monitor for a generic workload
type ScrapeInfo struct {
	// The path by which Prometheus should scrape metrics
	Path *string
	// The number of ports located for the workload
	Ports int
	// The basic authentication secret required for the service monitor if applicable
	BasicAuthSecret *corev1.Secret
	// Determines whether to enable Istio for the generated service monitor
	IstioEnabled *bool
	// Verify if the scrape target uses the Verrazzano Prometheus Labels
	VZPrometheusLabels *bool
	// The map to generate keep labels
	// This matches the expected pod labels to the scrape config
	KeepLabels map[string]string
	// The name of the cluster for the selected workload
	ClusterName string
}
