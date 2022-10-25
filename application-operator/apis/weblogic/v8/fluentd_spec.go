// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

import (
	corev1 "k8s.io/api/core/v1"
)

// FluentdSpecification defines the desired state of Domain
// +k8s:openapi-gen=true
type FluentdSpecification struct {
	// FluentdConfiguration, specify your own custom fluentd configuration.
	FluentdConfiguration string `json:"fluentdConfiguration,omitempty"`
	// Image, Fluentd container image name.
	Image string `json:"image,omitempty"`
	// ImagePullPolicy, image pull policy for the Fluentd sidecar container image.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Env, Array of environment variables to set in the fluentd container.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Resources, the requests and limits for the fluentd container.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	//VolumeMounts, Volume mounts for fluentd container.
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
	// OpenSearchCredentials, Fluentd open search credentials. A Kubernetes secret in the same namespace of the domain.
	OpenSearchCredentials string `json:"openSearchCredentials,omitempty"`
	// WatchIntrospectorLogs, if true Fluentd will watch introspector logs.
	WatchIntrospectorLogs bool `json:"watchIntrospectorLogs,omitempty"`
}
