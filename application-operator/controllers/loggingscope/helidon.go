// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	helidonWorkloadKey = "core.oam.dev/v1alpha2/ContainerizedWorkload"
	volumeVarlog       = "varlog"
	volumeData         = "datadockercontainers"
	confVolume         = "fluentd-config-volume"
	varLogPath         = "/var/log"
)

// helidonFluentdContainerConfigurationTemplate template for container specific FLUENTD rules
const helidonFluentdContainerConfigurationTemplate = `<source>
  @type tail
  path "/var/log/containers/#{ENV['WORKLOAD_NAME']}*{{ .ContainerName}}*.log"
  pos_file "/tmp/#{ENV['WORKLOAD_NAME']}-{{ .ContainerName}}.log.pos"
  read_from_head true
  tag {{ .WorkloadName}}-{{ .ContainerName}}
  # Helidon application messages are expected to look like this:
  # 2020.04.22 16:09:21 INFO org.books.bobby.Main Thread[main,5,main]: http://localhost:8080/books
  <parse>
    @type multi_format
    <pattern>
      # Docker output
      format json
      time_format %Y-%m-%dT%H:%M:%S.%NZ
    </pattern>
    <pattern>
      # cri-o output
      format regexp
      expression /^(?<timestamp>(.*?)) (?<stream>stdout|stderr) (?<log>.*)$/
      time_format %Y-%m-%dT%H:%M:%S.%N%:z
    </pattern>
  </parse>
</source>
<filter {{ .WorkloadName}}-{{ .ContainerName}}>
  @type record_transformer
  <record>
    oam.applicationconfiguration.namespace "#{ENV['NAMESPACE']}"
    oam.applicationconfiguration.name "#{ENV['APP_CONF_NAME']}"
    oam.component.namespace "#{ENV['NAMESPACE']}"
    oam.component.name  "#{ENV['COMPONENT_NAME']}"
    oam.container.name  "{{ .ContainerName}}"
    verrazzano.cluster.name  "#{ENV['CLUSTER_NAME']}"
  </record>
</filter>
<match {{ .WorkloadName}}-{{ .ContainerName}}>
  @type elasticsearch
  hosts "#{ENV['ELASTICSEARCH_URL']}"
  ca_file /fluentd/secret/ca-bundle
  user "#{ENV['ELASTICSEARCH_USER']}"
  password "#{ENV['ELASTICSEARCH_PASSWORD']}"
  index_name "#{ENV['NAMESPACE']}-#{ENV['APP_CONF_NAME']}-#{ENV['COMPONENT_NAME']}-{{ .ContainerName}}"
  include_timestamp true
  flush_interval 10s
</match>
`

// HelidonFluentdConfiguration FLUENTD rules for reading/parsing generic component log files
const HelidonFluentdConfiguration = `<label @FLUENT_LOG>
  <match fluent.*>
    @type stdout
  </match>
</label>
<filter **>
  @type parser
  key_name log
  <parse>
    @type grok
    <grok>
      name helidon-pattern
      pattern %{DATESTAMP:timestamp} %{DATA:loglevel} %{DATA:subsystem} %{DATA:thread} %{GREEDYDATA:message}
    </grok>
    <grok>
      name coherence-pattern
      pattern %{DATESTAMP:timestamp}/%{NUMBER:increment} %{DATA:subsystem} <%{DATA:loglevel}> (%{DATA:thread}): %{GREEDYDATA:message}
    </grok>
    <grok>
      name catchall-pattern
      pattern %{GREEDYDATA:message}
    </grok>
	time_key timestamp
	keep_time_key true
  </parse>
</filter>
`

// HelidonHandler injects FLUENTD sidecar container for generic Kubernetes Deployment
type HelidonHandler struct {
	client.Client
	Log logr.Logger
}

// Apply applies a logging scope to a Kubernetes Deployment
func (h *HelidonHandler) Apply(ctx context.Context, workload vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) (*ctrl.Result, error) {
	deploy, err := h.getDeployment(ctx, workload, scope)
	if err != nil {
		h.Log.Error(err, "Failed to fetch Deployment", "Deployment", workload.Name)
		return nil, err
	}
	// Apply logging to the in-memory child Deployment resource.
	result, err := h.ApplyToDeployment(ctx, workload, scope, deploy)
	if err != nil {
		h.Log.V(1).Info("Failed to apply logging to Deployment", "Deployment", deploy.Name, "error", err)
		return result, err
	}
	if result != nil {
		return result, nil
	}
	// Store the child Deployment resource.
	err = h.Update(ctx, deploy)
	if err != nil {
		h.Log.V(1).Info("Failed to update Deployment", "Deployment", deploy.Name, "error", err)
		return nil, err
	}
	return nil, nil
}

// ApplyToDeployment applies a logging scope to an existing in-memory Kubernetes Deployment
func (h *HelidonHandler) ApplyToDeployment(ctx context.Context, workload vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope, deploy *kapps.Deployment) (*ctrl.Result, error) {
	appContainers, fluentdFound := searchContainers(deploy.Spec.Template.Spec.Containers)
	h.Log.V(1).Info("Update Deployment", "Deployment", deploy.Name, "fluentdFound", fluentdFound)
	if fluentdFound {
		// If the Deployment does contain a FLUENTD container
		// requeue with a jittered delay to account for situation where the Deployment should be
		// updated by the oam-kubernetes-runtime
		duration := time.Duration(rand.IntnRange(5, 10)) * time.Second
		return &ctrl.Result{Requeue: true, RequeueAfter: duration}, nil
	}
	err := h.ensureFluentdConfigMap(ctx, scope.GetNamespace(), workload.Name, appContainers)
	if err != nil {
		return nil, err
	}
	err = ensureLoggingSecret(ctx, h, scope.GetNamespace(), &scope.Spec)
	if err != nil {
		return nil, err
	}
	volumes := CreateFluentdHostPathVolumes()
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volumes...)
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdConfigMapVolume(workload.Name))
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdSecretVolume(scope.Spec.SecretName))
	fluentdContainer := CreateFluentdContainer(scope.Spec, workload.Namespace, workload.Name)
	deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, fluentdContainer)
	return nil, nil
}

func searchContainers(containers []kcore.Container) ([]string, bool) {
	var appContainers []string
	fluentdFound := false
	for _, container := range containers {
		if container.Name == fluentdContainerName {
			fluentdFound = true
		} else {
			appContainers = append(appContainers, container.Name)
		}
	}
	return appContainers, fluentdFound
}

// Remove removes a logging scope from a Kubernetes Deployment
func (h *HelidonHandler) Remove(ctx context.Context, workload vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) (bool, error) {
	deploy, err := h.getDeployment(ctx, workload, scope)
	if err != nil {
		h.Log.Error(err, "Failed to fetch Deployment", "Deployment", workload.Name)
		return kerrs.IsNotFound(err), err
	}
	_, fluentdFound := searchContainers(deploy.Spec.Template.Spec.Containers)
	var errors []string
	if fluentdFound {
		err := h.deleteFluentdConfigMap(ctx, scope.GetNamespace(), workload.Name)
		if err != nil {
			errors = append(errors, err.Error())
		}
		existingValumes := deploy.Spec.Template.Spec.Volumes
		deploy.Spec.Template.Spec.Volumes = []kcore.Volume{}
		for _, vol := range existingValumes {
			if vol.Name != volumeVarlog && vol.Name != volumeData && vol.Name != confVolume && vol.Name != secretVolume {
				deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, vol)
			}
		}
		existingContainers := deploy.Spec.Template.Spec.Containers
		deploy.Spec.Template.Spec.Containers = []kcore.Container{}
		for _, container := range existingContainers {
			if container.Name != fluentdContainerName {
				deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, container)
			}
		}
		if err := h.Update(ctx, deploy); err != nil {
			errors = append(errors, err.Error())
		}
	}
	if errors != nil {
		return false, fmt.Errorf(strings.Join(errors, "\n"))
	}
	return true, nil
}

// getDeployment gets the Kubernetes Deployment
func (h *HelidonHandler) getDeployment(ctx context.Context, workload vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) (*kapps.Deployment, error) {
	deploy := &kapps.Deployment{}
	deploy.Namespace = scope.GetNamespace()
	deploy.Name = workload.Name
	depKey := client.ObjectKey{Name: workload.Name, Namespace: scope.GetNamespace()}
	if err := h.Get(ctx, depKey, deploy); err != nil {
		return nil, err
	}
	return deploy, nil
}

// CreateFluentdConfigMap creates the FLUENTD configmap for a given OAM application
func CreateFluentdConfigMap(namespace, name, fluentdConfig string) *kcore.ConfigMap {
	return &kcore.ConfigMap{
		ObjectMeta: kmeta.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: func() map[string]string {
			var data = make(map[string]string)
			data[fluentdConfKey] = fluentdConfig
			return data
		}(),
	}
}

// CreateFluentdContainer creates a FLUENTD sidecar container.
func CreateFluentdContainer(spec vzapi.LoggingScopeSpec, namespace, workloadName string) kcore.Container {
	container := kcore.Container{
		Name:            fluentdContainerName,
		Args:            []string{"-c", "/etc/fluent.conf"},
		Image:           spec.FluentdImage,
		ImagePullPolicy: kcore.PullIfNotPresent,
		Env: []kcore.EnvVar{
			{
				Name:  "WORKLOAD_NAME",
				Value: workloadName,
			},
			{
				Name:  "FLUENTD_CONF",
				Value: fluentdConfKey,
			},
			{
				Name:  "FLUENT_ELASTICSEARCH_SED_DISABLE",
				Value: "true",
			},
			{
				Name:  "NAMESPACE",
				Value: namespace,
			},
			{
				Name: "APP_CONF_NAME",
				ValueFrom: &kcore.EnvVarSource{
					FieldRef: &kcore.ObjectFieldSelector{
						FieldPath: "metadata.labels['app.oam.dev/name']",
					},
				},
			},
			{
				Name: "COMPONENT_NAME",
				ValueFrom: &kcore.EnvVarSource{
					FieldRef: &kcore.ObjectFieldSelector{
						FieldPath: "metadata.labels['app.oam.dev/component']",
					},
				},
			},
			{
				Name:  elasticSearchURLEnv,
				Value: spec.ElasticSearchURL,
			},
			{
				Name: "CLUSTER_NAME",
				ValueFrom: &kcore.EnvVarSource{
					SecretKeyRef: &kcore.SecretKeySelector{
						LocalObjectReference: kcore.LocalObjectReference{
							Name: spec.SecretName,
						},
						Key: constants.ClusterNameData,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
			{
				Name: elasticSearchUserEnv,
				ValueFrom: &kcore.EnvVarSource{
					SecretKeyRef: &kcore.SecretKeySelector{
						LocalObjectReference: kcore.LocalObjectReference{
							Name: spec.SecretName,
						},
						Key: constants.ElasticsearchUsernameData,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
			{
				Name: elasticSearchPwdEnv,
				ValueFrom: &kcore.EnvVarSource{
					SecretKeyRef: &kcore.SecretKeySelector{
						LocalObjectReference: kcore.LocalObjectReference{
							Name: spec.SecretName,
						},
						Key: constants.ElasticsearchPasswordData,
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
		},
		VolumeMounts: []kcore.VolumeMount{
			{
				MountPath: fluentdConfMountPath,
				Name:      confVolume,
				SubPath:   fluentdConfKey,
				ReadOnly:  true,
			},
			{
				MountPath: secretMountPath,
				Name:      secretVolume,
				ReadOnly:  true,
			},
			{
				MountPath: varLogPath,
				Name:      volumeVarlog,
				ReadOnly:  true,
			},
			{
				MountPath: "/u01/data/docker/containers",
				Name:      volumeData,
				ReadOnly:  true,
			},
		},
	}

	return container
}

// CreateFluentdHostPathVolumes creates hostPath volumes to access container logs.
func CreateFluentdHostPathVolumes() []kcore.Volume {
	return []kcore.Volume{
		{
			Name: volumeVarlog,
			VolumeSource: kcore.VolumeSource{
				HostPath: &kcore.HostPathVolumeSource{
					Path: "/var/log",
				},
			},
		},
		{
			Name: volumeData,
			VolumeSource: kcore.VolumeSource{
				HostPath: &kcore.HostPathVolumeSource{
					Path: "/u01/data/docker/containers",
				},
			},
		},
	}
}

// CreateFluentdConfigMapVolume create a config map volume for FLUENTD.
func CreateFluentdConfigMapVolume(workloadName string) kcore.Volume {
	return kcore.Volume{
		Name: confVolume,
		VolumeSource: kcore.VolumeSource{
			ConfigMap: &kcore.ConfigMapVolumeSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: fluentdConfigMapName(workloadName),
				},
				DefaultMode: func(mode int32) *int32 {
					return &mode
				}(420),
			},
		},
	}
}

// CreateFluentdSecretVolume create a secret volume for FLUENTD.
func CreateFluentdSecretVolume(secretName string) kcore.Volume {
	return kcore.Volume{
		Name: secretVolume,
		VolumeSource: kcore.VolumeSource{
			Secret: &kcore.SecretVolumeSource{
				SecretName: secretName},
		},
	}
}

// fluentdConfigMapName returns the name of a components FLUENTD config map
// This uses a different configmap name from other workload types.
// The workload name is included so there is a configmap per component.
func fluentdConfigMapName(workloadName string) string {
	return fmt.Sprintf("fluentd-config-helidon-%s", workloadName)
}

func (h *HelidonHandler) ensureFluentdConfigMap(ctx context.Context, namespace, workloadName string, appContainersNames []string) error {

	helidonFluentdConfiguration := HelidonFluentdConfiguration
	// add the container specific configuration
	for _, containerName := range appContainersNames {
		tmpl, err := template.New("fluentdContainer").Parse(helidonFluentdContainerConfigurationTemplate)
		if err != nil {
			return err
		}

		data := struct {
			WorkloadName  string
			ContainerName string
		}{workloadName, containerName}

		var buf bytes.Buffer
		err = tmpl.Execute(&buf, data)
		if err != nil {
			return err
		}
		helidonFluentdConfiguration += buf.String()
	}
	// check if configmap exists
	name := fluentdConfigMapName(workloadName)
	configMap := &kcore.ConfigMap{}
	err := h.Get(ctx, objKey(namespace, name), configMap)
	if kerrs.IsNotFound(err) {
		if err = h.Create(ctx, CreateFluentdConfigMap(namespace, name, helidonFluentdConfiguration), &client.CreateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (h *HelidonHandler) deleteFluentdConfigMap(ctx context.Context, namespace, workloadName string) error {
	name := fluentdConfigMapName(workloadName)
	configMap := &kcore.ConfigMap{}
	err := h.Get(ctx, objKey(namespace, name), configMap)
	if !kerrs.IsNotFound(err) || err == nil {
		return h.Delete(ctx, configMap)
	}
	return err
}

func objKey(namespace, name string) client.ObjectKey {
	return client.ObjectKey{Name: name, Namespace: namespace}
}
