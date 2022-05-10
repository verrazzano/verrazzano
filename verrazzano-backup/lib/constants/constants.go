// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

import (
	"time"
)

// general constants
const (
	CmdTimeout              = time.Second * 300
	VerrazzanoNameSpaceName = "verrazzano-system"
	VeleroNameSpace         = "velero"
	BackupOperation         = "backup"
	RestoreOperation        = "restore"
	Min                     = 10
	Max                     = 25
	DevKey                  = "dev"
	TruthString             = "true"
	FalseString             = "false"
)

// secret related constants
const (
	AwsAccessKeyString       = "aws_access_key_id"     //nolint:gosec //#gosec G101
	AwsSecretAccessKeyString = "aws_secret_access_key" //nolint:gosec //#gosec G101
)

// opensearch constants
const (
	OpenSearchURL                        = "http://127.0.0.1:9200"
	OSComponent                          = "opensearch"
	OpenSearchDataPodPrefix              = "system-es-data"
	OpenSearchDataPodContainerName       = "es-data"
	OpenSearchMasterPodContainerName     = "es-master"
	OpenSearchIngestLabel                = "system-es-ingest"
	HTTPContentType                      = "application/json"
	OpeSearchSnapShotRepoName            = "vzbackup"
	RetryCount                           = 50
	OpenSearchSnapShotSucess             = "SUCCESS"
	OpenSearchSnapShotInProgress         = "IN_PROGRESS"
	IngestDeploymentName                 = "vmi-system-es-ingest"
	IngestLabelSelector                  = "app=system-es-ingest"
	KibanaDeploymentName                 = "vmi-system-kibana"
	KibanaLabelSelector                  = "app=system-kibana"
	VMODeploymentName                    = "verrazzano-monitoring-operator"
	VMOLabelSelector                     = "k8s-app=verrazzano-monitoring-operator"
	DataStreamGreen                      = "GREEN"
	OpensearchKeystoreAccessKeyCmd       = "/usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.access_key" //nolint:gosec //#nosec G204
	OpensearchKeystoreSecretAccessKeyCmd = "/usr/share/opensearch/bin/opensearch-keystore add --stdin --force s3.client.default.secret_key" //nolint:gosec //#nosec G204
)
