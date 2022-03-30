// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

const (
	storageVolumeName   = "weblogic-domain-storage-volume"
	workloadType        = "weblogic"
	scratchVolMountPath = "/scratch"
)

// WlsFluentdParsingRules defines the FLUENTD parsing rules for WLS
const WlsFluentdParsingRules = `<match fluent.**>
  @type null
</match>
<source>
  @type tail
  path "#{ENV['SERVER_LOG_PATH']}"
  pos_file /tmp/server.log.pos
  read_from_head true
  tag server_log
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
	format13 / <(?<message>([\s\S]*?))>\s*$/
	time_key timestamp
  </parse>
</source>
<source>
  @type tail
  path "#{ENV['DOMAIN_LOG_PATH']}"
  pos_file /tmp/domain.log.pos
  read_from_head true
  tag domain_log
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
	format13 / <(?<message>([\s\S]*?))>\s*$/
	time_key timestamp
  </parse>
</source>
<source>
  @type tail
  path "#{ENV['ACCESS_LOG_PATH']}"
  pos_file /tmp/access.log.pos
  read_from_head true
  tag server_access_log
  <parse>
	@type none
  </parse>
</source>
<source>
  @type tail
  path "#{ENV['NODEMANAGER_LOG_PATH']}"
  pos_file /tmp/nodemanager.log.pos
  read_from_head true
  tag server_nodemanager_log
  <parse>
	@type none
  </parse>
</source>
<filter **>
  @type record_transformer
  <record>
    domainUID "#{ENV['DOMAIN_UID']}"
    oam.applicationconfiguration.namespace "#{ENV['NAMESPACE']}"
    oam.applicationconfiguration.name "#{ENV['APP_CONF_NAME']}"
    oam.component.namespace "#{ENV['NAMESPACE']}"
    oam.component.name "#{ENV['COMPONENT_NAME']}"
    verrazzano.cluster.name "#{ENV['CLUSTER_NAME']}"
  </record>
</filter>
<filter server_log>
  @type record_transformer
  <record>
    wls_log_stream "server_log"
  </record>
</filter>
<filter domain_log>
  @type record_transformer
  <record>
    wls_log_stream "domain_log"
  </record>
</filter>
<filter server_access_log>
  @type record_transformer
  <record>
    wls_log_stream "server_access_log"
  </record>
</filter>
<filter server_nodemanager_log>
  @type record_transformer
  <record>
    wls_log_stream "server_nodemanager_log"
  </record>
</filter>
<match **>
  @type stdout
</match>
`

// getWlsSpecificContainerEnv builds WLS specific env vars
func getWlsSpecificContainerEnv(name string) []v1.EnvVar {
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
		{
			Name:  "SERVER_LOG_PATH",
			Value: getWLSServerLogPath(name),
		},
		{
			Name:  "ACCESS_LOG_PATH",
			Value: getWLSServerAccessLogPath(name),
		},
		{
			Name:  "NODEMANAGER_LOG_PATH",
			Value: getWLSServerNodeManagerPath(name),
		},
		{
			Name:  "DOMAIN_LOG_PATH",
			Value: getWLSDomainLogPath(name),
		},
	}
}

func getWLSLogPath(name string) string {
	return getWLSServerLogPath(name) + "," + getWLSServerAccessLogPath(name) + "," + getWLSServerNodeManagerPath(name) + "," + getWLSDomainLogPath(name)
}

func getWLSServerLogPath(name string) string {
	return fmt.Sprintf("%s/$(SERVER_NAME).log", getWLSLogHome(name))
}

func getWLSServerAccessLogPath(name string) string {
	return fmt.Sprintf("%s/$(SERVER_NAME)_access.log", getWLSLogHome(name))
}

func getWLSServerNodeManagerPath(name string) string {
	return fmt.Sprintf("%s/$(SERVER_NAME)_nodemanager.log", getWLSLogHome(name))
}

func getWLSDomainLogPath(name string) string {
	return fmt.Sprintf("%s/$(DOMAIN_UID).log", getWLSLogHome(name))
}

// getWLSLogHome builds a log home give a resource name
func getWLSLogHome(name string) string {
	return fmt.Sprintf("%s/logs/%s", scratchVolMountPath, name)
}
