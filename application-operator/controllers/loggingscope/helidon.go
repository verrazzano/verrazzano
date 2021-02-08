// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	volumeConf         = "fluentd-config-volume"
)

// HelidonFluentdConfiguration FLUENTD rules for reading/parsing generic component log files
const HelidonFluentdConfiguration = `<label @FLUENT_LOG>
  <match fluent.*>
    @type stdout
  </match>
</label>
<source>
  @type tail
  path "/var/log/containers/#{ENV['WORKLOAD_NAME']}*#{ENV['APP_CONTAINER_NAME']}*.log"
  pos_file "/tmp/#{ENV['WORKLOAD_NAME']}.log.pos"
  read_from_head true
  tag "#{ENV['WORKLOAD_NAME']}"
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
<filter **>
  @type record_transformer
  <record>
    oam.applicationconfiguration.namespace "#{ENV['NAMESPACE']}"
    oam.applicationconfiguration.name "#{ENV['APP_CONF_NAME']}"
    oam.component.namespace "#{ENV['NAMESPACE']}"
    oam.component.name  "#{ENV['COMPONENT_NAME']}"
  </record>
</filter>
<match **>
  @type elasticsearch
  host "#{ENV['ELASTICSEARCH_HOST']}"
  port "#{ENV['ELASTICSEARCH_PORT']}"
  user "#{ENV['ELASTICSEARCH_USER']}"
  password "#{ENV['ELASTICSEARCH_PASSWORD']}"
  index_name "` + ElasticSearchIndex + `"
  scheme http
  include_timestamp true
  flush_interval 10s
</match>
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
	appContainer, fluentdFound := searchContainers(deploy.Spec.Template.Spec.Containers)
	h.Log.V(1).Info("Update Deployment", "Deployment", deploy.Name, "fluentdFound", fluentdFound)
	if fluentdFound {
		// If the Deployment does contain a FLUENTD container
		// requeue with a jittered delay to account for situation where the Deployment should be
		// updated by the oam-kubernetes-runtime
		duration := time.Duration(rand.IntnRange(5, 10)) * time.Second
		return &ctrl.Result{Requeue: true, RequeueAfter: duration}, nil
	}
	err = h.ensureFluentdConfigMap(ctx, scope.GetNamespace(), workload.Name)
	if err != nil {
		return nil, err
	}
	err = h.ensureEsSecret(ctx, scope.GetNamespace(), scope.Spec.SecretName)
	if err != nil {
		return nil, err
	}
	volumes := CreateFluentdHostPathVolumes()
	for _, volume := range volumes {
		deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, volume)
	}
	deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, CreateFluentdConfigMapVolume(workload.Name))
	fluentdContainer := CreateFluentdContainer(workload.Namespace, workload.Name, appContainer, scope.Spec.FluentdImage, scope.Spec.SecretName, scope.Spec.ElasticSearchHost)
	deploy.Spec.Template.Spec.Containers = append(deploy.Spec.Template.Spec.Containers, fluentdContainer)
	err = h.Update(ctx, deploy)
	if err != nil {
		h.Log.V(1).Info("Failed to update Deployment", "Deployment", deploy.Name, "error", err)
		return nil, err
	}
	return nil, nil
}

func searchContainers(containers []kcore.Container) (string, bool) {
	var appContainer string
	fluentdFound := false
	for _, container := range containers {
		if container.Name == fluentdContainerName {
			fluentdFound = true
		} else {
			appContainer = container.Name
		}
	}
	return appContainer, fluentdFound
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
			if vol.Name != volumeVarlog && vol.Name != volumeData && vol.Name != volumeConf {
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
			var data map[string]string
			data = make(map[string]string)
			data[fluentdConfKey] = fluentdConfig
			return data
		}(),
	}
}

// CreateFluentdContainer creates a FLUENTD sidecar container.
func CreateFluentdContainer(namespace, workloadName, containerName, fluentdImage, esSecret, esHost string) kcore.Container {
	container := kcore.Container{
		Name:            fluentdContainerName,
		Args:            []string{"-c", "/etc/fluent.conf"},
		Image:           fluentdImage,
		ImagePullPolicy: kcore.PullIfNotPresent,
		Env: []kcore.EnvVar{
			{
				Name:  "WORKLOAD_NAME",
				Value: workloadName,
			},
			{
				Name:  "APP_CONTAINER_NAME",
				Value: containerName,
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
				Name:  "ELASTICSEARCH_HOST",
				Value: esHost,
			},
			{
				Name:  "ELASTICSEARCH_PORT",
				Value: "9200",
			},
			{
				Name: "ELASTICSEARCH_USER",
				ValueFrom: &kcore.EnvVarSource{
					SecretKeyRef: &kcore.SecretKeySelector{
						LocalObjectReference: kcore.LocalObjectReference{
							Name: esSecret,
						},
						Key: "username",
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
			{
				Name: "ELASTICSEARCH_PASSWORD",
				ValueFrom: &kcore.EnvVarSource{
					SecretKeyRef: &kcore.SecretKeySelector{
						LocalObjectReference: kcore.LocalObjectReference{
							Name: esSecret,
						},
						Key: "password",
						Optional: func(opt bool) *bool {
							return &opt
						}(true),
					},
				},
			},
		},
		VolumeMounts: []kcore.VolumeMount{
			{
				MountPath: "/fluentd/etc/fluentd.conf",
				Name:      volumeConf,
				SubPath:   fluentdConfKey,
				ReadOnly:  true,
			},
			{
				MountPath: "/var/log",
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
		Name: volumeConf,
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

// fluentdConfigMapName returns the name of a components FLUENTD config map
func fluentdConfigMapName(workloadName string) string {
	return fmt.Sprintf("%s-fluentd", workloadName)
}

func replicateVmiSecret(vmiSec *kcore.Secret, namespace, name string) *kcore.Secret {
	sec := &kcore.Secret{
		ObjectMeta: kmeta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: vmiSec.Data,
	}
	return sec
}

func (h *HelidonHandler) ensureFluentdConfigMap(ctx context.Context, namespace, workloadName string) error {
	// check if configmap exists
	name := fluentdConfigMapName(workloadName)
	configMap := &kcore.ConfigMap{}
	err := h.Get(ctx, objKey(namespace, name), configMap)
	if kerrs.IsNotFound(err) {
		if err = h.Create(ctx, CreateFluentdConfigMap(namespace, name, HelidonFluentdConfiguration), &client.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}
	return err
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

func (h *HelidonHandler) ensureEsSecret(ctx context.Context, namespace, name string) error {
	secret := &kcore.Secret{}
	err := h.Get(ctx, objKey(namespace, name), secret)
	if kerrs.IsNotFound(err) {
		secretKey := client.ObjectKey{Name: "verrazzano", Namespace: "verrazzano-system"}
		err = h.Get(ctx, secretKey, secret)
		if err != nil {
			return err
		}
		secret = replicateVmiSecret(secret, namespace, name)
		if err = h.Create(ctx, secret, &client.CreateOptions{}); err != nil {
			return err
		}
		return nil
	}
	return err
}

func objKey(namespace, name string) client.ObjectKey {
	return client.ObjectKey{Name: name, Namespace: namespace}
}
