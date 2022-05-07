// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import (
	"time"
)

const (
	CmdTimeout              = time.Second * 300
	VerrazzanoNameSpaceName = "verrazzano-system"
	VeleroNameSpace         = "velero"
	BackupOperation         = "backup"
	RestoreOperation        = "restore"
	Min                     = 10
	Max                     = 25
)

//opensearch constants
const (
	EsUrl                                = "http://127.0.0.1:9200"
	OSComponent                          = "opensearch"
	OpenSearchDataPodPrefix              = "system-es-data"
	OpenSearchDataPodContainerName       = "es-data"
	OpenSearchMasterPodContainerName     = "es-master"
	OpenSearchIngestLabel                = "system-es-ingest"
	HTTPContentType                      = "application/json"
	OpeSearchSnapShotRepoName            = "vzbackup"
	SnapshotRetryCount                   = 20
	OpenSearchSnapShotSucess             = "SUCCESS"
	OpenSearchSnapShotInProgress         = "IN_PROGRESS"
	IngestDeploymentName                 = "vmi-system-es-ingest"
	IngestLabelSelector                  = "app=system-es-ingest"
	VMODeploymentName                    = "verrazzano-monitoring-operator"
	VMOLabelSelector                     = "k8s-app=verrazzano-monitoring-operator"
	DataStreamGreen                      = "GREEN"
	OpensearchKeystoreAccessKeyCmd       = "/usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.access_key" //#nosec G204
	OpensearchkeystoreSecretAccessKeyCmd = "/usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.secret_key" //#nosec G204
)
