// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v8

import (
	corev1 "k8s.io/api/core/v1"
)

// DomainSpec defines the desired state of Domain
// +k8s:openapi-gen=true
type DomainSpec struct {
	// AdminServer contains configuration for the admin server
	// Note: this value is required by WebLogic Operator, but is marked optional because Verrazzano can provide a
	// default value.
	AdminServer AdminServer `json:"adminServer,omitempty"`

	// AllowReplicasBelowMinDynClusterSize determines whether to allow the number of running cluster member Managed Server instances to drop below the minimum
	// dynamic cluster size configured in the WebLogic domain configuration, if this is not specified for a specific
	// cluster under the clusters field. Defaults to true.
	AllowReplicasBelowMinDynClusterSize bool `json:"allowReplicasBelowMinDynClusterSize,omitempty"`

	// Configure auxiliary image volumes including their respective mount paths. Auxiliary image volumes are in turn referenced by one or more
	// serverPod.auxiliaryImages mounts, and are internally implemented using a Kubernetes emptyDir volume.
	// +x-kubernetes-list-type=set
	AuxiliaryImageVolumes []AuxiliaryImageVolume `json:"auxiliaryImageVolumes,omitempty"`

	// Clusters contains configuration for the clusters
	// +x-kubernetes-list-type=set
	Clusters []Cluster `json:"clusters,omitempty"`

	// ConfigOverrides contains the name of the config map for optional WebLogic configuration overrides
	ConfigOverrides string `json:"configOverrides,omitempty"`

	// ConfigOverrideSecrets contains a list of names of the secrets for optional WebLogic configuration overrides
	// +x-kubernetes-list-type=set
	ConfigOverrideSecrets []string `json:"configOverrideSecrets,omitempty"`

	// Configuration contains configuration options for the WebLogic Kubernetes Operator
	Configuration Configuration `json:"configuration,omitempty"`

	// DataHome An optional directory in a server's container for data storage of default and custom file stores. If dataHome
	// is not specified or its value is either not set or empty, then the data storage directories are determined
	// from the WebLogic domain configuration.
	DataHome string `json:"dataHome,omitempty"`

	// DomainHome The folder for the WebLogic Domain
	DomainHome string `json:"domainHome,omitempty"`

	// DomainHomeInImage True if this domain's home is defined in the docker image for the domain
	// Note: this value is required by WebLogic Operator, but is marked optional because Verrazzano can provide a
	// default value.
	DomainHomeInImage bool `json:"domainHomeInImage,omitempty"`

	// DomainHomeSourceType Domain home file system source type: Legal values: Image, PersistentVolume, FromModel. Image indicates that
	// the domain home file system is present in the container image specified by the image field. PersistentVolume
	// indicates that the domain home file system is located on a persistent volume. FromModel indicates that the
	// domain home file system will be created and managed by the operator based on a WDT domain model. If this
	// field is specified, it overrides the value of domainHomeInImage. If both fields are unspecified, then
	// domainHomeSourceType defaults to Image.
	DomainHomeSourceType string `json:"domainHomeSourceType,omitempty"`

	// DomainUID The name of the WebLogic domain
	DomainUID string `json:"domainUID,omitempty"`

	// FluentdSpecification The Fluentd specification for sidecar logging
	FluentdSpecification FluentdSpecification `json:"fluentdSpecification,omitempty"`

	// HTTPAccessLogInLogHome specifies whether the server HTTP access log files will be written to the same directory specified in logHome.
	// Otherwise, server HTTP access log files will be written to the directory configured in the WebLogic domain
	// configuration. Defaults to true.
	HTTPAccessLogInLogHome bool `json:"httpAccessLogInLogHome,omitempty"`

	// Image The WebLogic Docker image; required when domainHomeInImage is true
	Image string `json:"image"`

	// ImagePullPolicy The image pull policy for the WebLogic Docker image
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets A list of image pull secrets for the WebLogic Docker image
	// +x-kubernetes-list-type=set
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets"`

	// IncludeServerOutInPodLog If true (the default), the server .out file will be included in the pod's stdout.
	IncludeServerOutInPodLog bool `json:"includeServerOutInPodLog,omitempty"`

	// IntrospectVersion Changes to this field cause the operator to repeat its introspection of the WebLogic domain configuration.
	IntrospectVersion string `json:"introspectVersion,omitempty"`

	// LivenessProbeCustomScript Full path of an optional liveness probe custom script for WebLogic Server instance
	// pods. The existing liveness probe script livenessProbe.sh will invoke this custom script after the existing
	// script performs its own checks. This element is optional and is for advanced usage only. Its value is not set by
	// default. If the custom script fails with non-zero exit status, then pod will fail the liveness probe and
	// Kubernetes will restart the container. If the script specified by this element value is not found, then it
	// is ignored.
	LivenessProbeCustomScript string `json:"livenessProbeCustomScript,omitempty"`

	// LogHome The in-pod name of the directory in which to store the domain, node manager, server logs, and server *.out files
	LogHome string `json:"logHome"`

	// LogHomeEnabled Specified whether the log home folder is enabled
	LogHomeEnabled bool `json:"logHomeEnabled,omitempty"`

	// ManagedServers Configuration for individual Managed Servers
	// +x-kubernetes-list-type=set
	ManagedServers []ManagedServer `json:"managedServers,omitempty"`

	// MaxClusterConcurrentShutdown The default maximum number of WebLogic Server instances that a cluster will shut
	// down in parallel when it is being partially shut down by lowering its replica count. You can override this
	// default on a per cluster basis by setting the cluster's maxConcurrentShutdown field. A value of 0 means there is
	// no limit. Defaults to 1.
	MaxClusterConcurrentShutdown int32 `json:"maxClusterConcurrentShutdown,omitempty"`

	// MaxClusterConcurrentStartup The maximum number of cluster member Managed Server instances that the operator will start in parallel
	// for a given cluster, if maxConcurrentStartup is not specified for a specific cluster under the clusters field.
	// A value of 0 means there is no configured limit. Defaults to 0.
	MaxClusterConcurrentStartup int32 `json:"maxClusterConcurrentStartup,omitempty"`

	// MonitoringExporter Automatic deployment and configuration of the WebLogic Monitoring Exporter. If specified, the
	// operator will deploy a sidecar container alongside each WebLogic Server instance that runs the exporter. WebLogic
	// Server instances that are already running when the monitoringExporter field is created or deleted, will not be
	// affected until they are restarted. When any given server is restarted for another reason, such as a change to the
	// restartVersion, then the newly created pod will have the exporter sidecar or not, as appropriate.
	// See https://github.com/oracle/weblogic-monitoring-exporter.
	MonitoringExporter MonitoringExporterSpec `json:"monitoringExporter,omitempty"`

	// Replicas The number of managed servers to run in any cluster that does not specify a replica count.
	// This is a pointer to distinguish between explicit zero and not specified.
	// Note: this value is required by WebLogic Operator, but is marked optional because Verrazzano can provide a default value.
	Replicas *int32 `json:"replicas,omitempty"`

	// RestartVersion If present, every time this value is updated the operator will restart
	// the required servers.
	RestartVersion string `json:"restartVersion,omitempty"`

	// ServerPod describes the pod where a WebLogic server will run
	ServerPod ServerPod `json:"serverPod,omitempty"`

	// ServerService Customization affecting ClusterIP Kubernetes services for WebLogic Server instances.
	ServerService ServerService `json:"serverService,omitempty"`

	// ServerStartPolicy The strategy for deciding whether to start a server.  Legal values are ADMIN_ONLY, NEVER, or IF_NEEDED.
	// Note: this value is required by WebLogic Operator, but is marked optional because Verrazzano can provide a
	// default value.
	ServerStartPolicy string `json:"serverStartPolicy,omitempty"`

	// ServerStartState The state in which the server is to be started.  Legal values are "RUNNING" or "ADMIN"
	ServerStartState string `json:"serverStartState,omitempty"`

	// WebLogicCredentialsSecret The name of a pre-created Kubernetes secret, in the domain's namepace, that holds the username and password
	// needed to boot WebLogic Server
	WebLogicCredentialsSecret corev1.SecretReference `json:"webLogicCredentialsSecret"`
}
