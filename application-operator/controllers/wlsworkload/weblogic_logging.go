// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"

	v1 "k8s.io/api/core/v1"
)

const (
	storageVolumeName      = "weblogic-domain-storage-volume"
	storageVolumeMountPath = logging.ScratchVolMountPath
	workloadType           = "weblogic"
)

// WlsFluentdParsingRules defines the FLUENTD parsing rules for WLS
const WlsFluentdParsingRules = `<match fluent.**>
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
	format13 / <(?<message>[^>]*)>[\s]*/
	time_key timestamp
	keep_time_key true
  </parse>
</source>
<filter **>
  @type record_transformer
  <record>
    domainUID "#{ENV['DOMAIN_UID']}"
    oam.applicationconfiguration.namespace "#{ENV['NAMESPACE']}"
    oam.applicationconfiguration.name "#{ENV['APP_CONF_NAME']}"
    oam.component.namespace "#{ENV['NAMESPACE']}"
    oam.component.name  "#{ENV['COMPONENT_NAME']}"
    verrazzano.cluster.name  "#{ENV['CLUSTER_NAME']}"
  </record>
</filter>
<match **>
  @type stdout
</match>
`

// GetWlsSpecificContainerEnv builds WLS specific env vars
func GetWlsSpecificContainerEnv() []v1.EnvVar {
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

// BuildWLSLogPath builds a log path given a resource name
func BuildWLSLogPath(name string) string {
	return fmt.Sprintf("%s/logs/%s/$(SERVER_NAME).log", logging.ScratchVolMountPath, name)
}

// BuildWLSLogHome builds a log home give a resource name
func BuildWLSLogHome(name string) string {
	return fmt.Sprintf("%s/logs/%s", logging.ScratchVolMountPath, name)
}
