// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	wls "github.com/verrazzano/verrazzano-crd-generator/pkg/apis/weblogic/v8"
	vzapi "github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	wlsDomainKind     = "Domain"
	storageVolumeName = "weblogic-domain-storage-volume"
)

// FLUENTD parsing rules for WLS
const wlsFluentdParsingRules = `<match fluent.**>
  @type null
</match>
<source>
  @type tail
  path "#{ENV['LOG_PATH']}"
  pos_file /tmp/server.log.pos
  read_from_head true
  tag "#{ENV['DOMAIN_UID']}"
  # messages look like this:
  #   firstline:  ####
  #   format1:    <Mar 17, 2020 2:41:55,029 PM EDT> 
  #   format2:    <Info> 
  #   format3:    <WorkManager>
  #   format4:    <meerkat>
  #   format5:    <AdminServer>
  #   format6:    <Timer-2> 
  #   format7:    <<WLS Kernel>> 
  #   format8:    <> 
  #   format9:    <00ccb822-8beb-4ce0-905d-2039c4fd676f-00000010> 
  #   format10:   <1584470515029> 
  #   format11:   <[severity-value: 64] [rid: 0] [partition-id: 0] [partition-name: DOMAIN] > 
  #   format12:   <BEA-002959> 
  #   formart13:  <Self-tuning thread pool contains 0 running threads, 1 idle threads, and 12 standby threads> 
  <parse>
	@type multiline
	format_firstline /^####/
	format1 /^####<(?<timestamp>(.*?))>/
	format2 / <(?<level>(.*?))>/
	format3 / <(?<subSystem>(.*?))>/
	format4 / <(?<serverName>(.*?))>/
	format5 / <(?<serverName2>(.*?))>/
	format6 / <(?<threadName>(.*?))>/
	format7 / <(?<info1>(.*?))>/
	format8 / <(?<info2>(.*?))>/
	format9 / <(?<info3>(.*?))>/
	format10 / <(?<sequenceNumber>(.*?))>/
	format11 / <(?<severity>(.*?))>/
	format12 / <(?<messageID>(.*?))>/
	format13 / <(?<message>(.*?))>/
	time_key timestamp
	keep_time_key true
  </parse>
</source>
<filter **>
  @type record_transformer
  <record>
    domainUID "#{ENV['DOMAIN_UID']}"
  </record>
</filter>
<match **>
  @type elasticsearch
  host "#{ENV['ELASTICSEARCH_HOST']}"
  port "#{ENV['ELASTICSEARCH_PORT']}"
  user "#{ENV['ELASTICSEARCH_USER']}"
  password "#{ENV['ELASTICSEARCH_PASSWORD']}"
  index_name "#{ENV['DOMAIN_UID']}"
  scheme http
  key_name timestamp 
  types timestamp:time
  include_timestamp true
</match>
`

var getFluentdManager = getFluentd

// wlsHandler handles FLUENTD integration for WLS domains
type wlsHandler struct {
	k8sclient.Client
	Log logr.Logger
}

// Apply applies a logging scope to a WLS Domain
func (h *wlsHandler) Apply(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) error {
	name := resource.Name
	domain := createWlsDomain(resource)

	// get the corresponding domain
	key, _ := k8sclient.ObjectKeyFromObject(&domain)
	err := h.Get(ctx, key, &domain)
	if err != nil {
		return err
	}
	serverPod := domain.Spec.ServerPod
	fluentdPod := toFluentdPod(serverPod, buildWLSLogPath(name))
	updated, err := getFluentdManager(ctx, h.Log, h).Apply(scope, resource, fluentdPod)

	if updated && err == nil {
		serverPod.Containers = fluentdPod.Containers
		serverPod.Volumes = fluentdPod.Volumes
		serverPod.VolumeMounts = fluentdPod.VolumeMounts
		domain.Spec.ServerPod = serverPod
		domain.Spec.Configuration.Istio.Enabled = false
		domain.Spec.LogHome = buildWLSLogHome(name)
		domain.Spec.LogHomeEnabled = true

		err = h.Update(ctx, &domain)

	}
	return err
}

// Remove removes a logging scope from a WLS Domain
func (h *wlsHandler) Remove(ctx context.Context, resource vzapi.QualifiedResourceRelation, scope *vzapi.LoggingScope) (bool, error) {
	domain := createWlsDomain(resource)
	// get the corresponding domain
	key, _ := k8sclient.ObjectKeyFromObject(&domain)
	err := h.Get(ctx, key, &domain)
	if err != nil {
		h.Log.Info("Unable to lookup domain. Assuming that it has been deleted", "domain", domain)
	}

	fluentdPod := toFluentdPod(domain.Spec.ServerPod, "")
	// indicates whether or not we have confirmed that all remove related changes have been made in the system
	removeVerified := getFluentdManager(ctx, h.Log, h).Remove(scope, resource, fluentdPod)

	if !removeVerified {
		domain.Spec.ServerPod.Volumes = fluentdPod.Volumes
		domain.Spec.ServerPod.VolumeMounts = fluentdPod.VolumeMounts
		domain.Spec.ServerPod.Containers = fluentdPod.Containers
		err = h.Update(ctx, &domain)
	}

	return removeVerified, err
}

// getFluentd creates an instance of FluentManager
func getFluentd(ctx context.Context, log logr.Logger, client k8sclient.Client) FluentdManager {
	return &fluentd{Context: ctx, Log: log, Client: client, ParseRules: wlsFluentdParsingRules, StorageVolumeName: storageVolumeName}
}

// createWlsDomain creates a WLS Domain instance
func createWlsDomain(resource vzapi.QualifiedResourceRelation) wls.Domain {
	return wls.Domain{TypeMeta: metav1.TypeMeta{Kind: wlsDomainKind, APIVersion: resource.APIVersion},
		ObjectMeta: metav1.ObjectMeta{Name: resource.Name, Namespace: resource.Namespace}}
}

// toFluentdPod creates a FluentdPod instance from a WLS ServerPod
func toFluentdPod(serverPod wls.ServerPod, logPath string) *FluentdPod {
	return &FluentdPod{
		Containers:   serverPod.Containers,
		Volumes:      serverPod.Volumes,
		VolumeMounts: serverPod.VolumeMounts,
		LogPath:      logPath,
		HandlerEnv:   getWlsSpecificContainerEnv(),
	}
}

// getWlsSpecificContainerEnv builds WLS specific env vars
func getWlsSpecificContainerEnv() []v1.EnvVar {
	return []v1.EnvVar{
		{
			Name: "DOMAIN_UID",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.labels['weblogic.domainUID']",
				},
			},
		},
		{
			Name: "SERVER_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.labels['weblogic.serverName']",
				},
			},
		},
	}
}

// buildWLSLogPath builds a log path given a resource name
func buildWLSLogPath(name string) string {
	return fmt.Sprintf("/scratch/logs/%s/$(SERVER_NAME).log", name)
}

// buildWLSLogHome builds a log home give a resource name
func buildWLSLogHome(name string) string {
	return fmt.Sprintf("/scratch/logs/%s", name)
}
