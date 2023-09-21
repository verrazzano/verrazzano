// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

const (
	ObjectStoreCredsAccessKeyName       = "aws_access_key_id"     //nolint:gosec //#gosec G101 //#gosec G204
	ObjectStoreCredsSecretAccessKeyName = "aws_secret_access_key" //nolint:gosec //#gosec G101 //#gosec G204
	RancherUserCount                    = 1
	KeycloakUserCount                   = 3
	BackupResource                      = "backups"
	RestoreResource                     = "restores"
	BackupStorageLocationResource       = "backupstoragelocations"
	BackupPodVolumeResource             = "podvolumebackups"
	RestorePodVolumeResource            = "podvolumerestores"
	InnoDBClusterName                   = "mysql"
)

// SecretsData template for creating backup credentials
const SecretsData = //nolint:gosec //#gosec G101 //#gosec G204
`[default]
{{ .AccessName }}={{ .ObjectStoreAccessValue }}
{{ .ScrtName }}={{ .ObjectStoreScrt }}
`

// ProfileData template for creating backup credentials
const ProfileData = `[default]
region={{ .Region }}
`

// VeleroBackupLocation template for creating velero backup storage location object.
const VeleroBackupLocation = `
    apiVersion: velero.io/v1
    kind: BackupStorageLocation
    metadata:
      name: {{ .VeleroBackupStorageName }}
      namespace: {{ .VeleroNamespaceName }}
    spec:
      provider: aws
      objectStorage:
        bucket: {{ .VeleroObjectStoreBucketName }}
        prefix: {{ .VeleroBackupStorageName }}
      credential:
        name: {{ .VeleroSecretName }}
        key: cloud
      config:
        region: {{ .VeleroBackupRegion }}
        s3ForcePathStyle: "true"
        s3Url: https://{{ .VeleroObjectStorageNamespaceName }}.compat.objectstorage.{{ .VeleroBackupRegion }}.oraclecloud.com`

// VeleroBackup template for creating velero backup object.
const VeleroBackup = `
---
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: {{ .VeleroBackupName }}
  namespace: {{ .VeleroNamespaceName }}
spec:
  includedNamespaces:
    - verrazzano-logging
  labelSelector:
    matchLabels:
      opster.io/opensearch-cluster: opensearch
  defaultVolumesToRestic: false
  storageLocation: {{ .VeleroBackupStorageName }}
  hooks:
    resources:
      - 
        name: {{ .VeleroOpensearchHookResourceName }}
        includedNamespaces:
          - verrazzano-logging
        labelSelector:
          matchLabels:
            statefulset.kubernetes.io/pod-name: opensearch-es-master-0
        post:
          - 
            exec:
              container: opensearch
              command:
                - /usr/share/opensearch/bin/verrazzano-backup-hook
                - -operation
                - backup
                - -velero-backup-name
                - {{ .VeleroBackupName }}
              onError: Fail
              timeout: 10m`

// VeleroRestore template for creating velero restore object.
const VeleroRestore = `
---
apiVersion: velero.io/v1
kind: Restore
metadata:
  name: {{ .VeleroRestore }}
  namespace: {{ .VeleroNamespaceName }}
spec:
  backupName: {{ .VeleroBackupName }}
  includedNamespaces:
    - verrazzano-logging
  labelSelector:
    matchLabels:
      opster.io/opensearch-cluster: opensearch
  restorePVs: false
  hooks:
    resources:
      - name: {{ .VeleroOpensearchHookResourceName }}
        includedNamespaces:
          - verrazzano-logging
        labelSelector:
          matchLabels:
            statefulset.kubernetes.io/pod-name: opensearch-es-master-0
        postHooks:
		  - init:
              timeout: 30m
              initContainers:
                - args:
                    - /usr/share/opensearch/bin/verrazzano-backup-hook --operation=pre-restore --velero-backup-name={{ .VeleroBackupName }}
                  command:
                    - sh
                    - -c
                  image: iad.ocir.io/odsbuilddev/sandboxes/saket.m.mahto/opensearch-security:latest
                  imagePullPolicy: Always
                  name: pre-hook
          - exec:
              container: opensearch
              command:
                - /usr/share/opensearch/bin/verrazzano-backup-hook
                - -operation
                - restore
                - -velero-backup-name
                - {{ .VeleroBackupName }}
              waitTimeout: 30m
              execTimeout: 30m
              onError: Fail`

// EsQueryBody template for opensearch query
const EsQueryBody = `
{
	"query": {
  		"terms": {
			"_id": ["{{ .BackupIDBeforeBackup }}"]
  		}
	}
}
`

// RancherUserTemplate template body for creating rancher test user
const RancherUserTemplate = `
{
  "description":"Automated Tests", 
  "mustChangePassword":false, 
  "enabled": true,
  "name": {{ .FullName }}, 
  "password": {{ .Password }}, 
  "username": {{ .Username }}
}
`

// RancherBackup template for creating rancher backup object.
const RancherBackup = `
---
apiVersion: resources.cattle.io/v1
kind: Backup
metadata:
  name: {{ .RancherBackupName }}
spec:
  storageLocation:
    s3:
      credentialSecretName: {{ .RancherSecretData.RancherSecretName }}
      credentialSecretNamespace: {{ .RancherSecretData.RancherSecretNamespaceName }}
      bucketName: {{ .RancherSecretData.RancherObjectStoreBucketName }}
      folder: rancher-backup
      region: {{ .RancherSecretData.RancherBackupRegion }}
      endpoint: {{ .RancherSecretData.RancherObjectStorageNamespaceName }}.compat.objectstorage.{{ .RancherSecretData.RancherBackupRegion }}.oraclecloud.com
  resourceSetName: rancher-resource-set
`

// RancherRestore template for creating rancher restore object.
const RancherRestore = `
---
apiVersion: resources.cattle.io/v1
kind: Restore
metadata:
  name: {{ .RancherRestoreName }}
spec:
  backupFilename: {{ .BackupFileName }}
  storageLocation:
    s3:
      credentialSecretName: {{ .RancherSecretData.RancherSecretName }}
      credentialSecretNamespace: {{ .RancherSecretData.RancherSecretNamespaceName }}
      bucketName: {{ .RancherSecretData.RancherObjectStoreBucketName }}
      folder: rancher-backup
      region: {{ .RancherSecretData.RancherBackupRegion }}
      endpoint: {{ .RancherSecretData.RancherObjectStorageNamespaceName }}.compat.objectstorage.{{ .RancherSecretData.RancherBackupRegion }}.oraclecloud.com
`

const VeleroMySQLBackup = `
---
apiVersion: velero.io/v1
kind: Backup
metadata:
  name: {{ .VeleroMysqlBackupName }}
  namespace: {{ .VeleroNamespaceName }}
spec:
  includedNamespaces:
    - keycloak  
  defaultVolumesToRestic: true
  storageLocation: {{ .VeleroMysqlBackupStorageName }}
  hooks:
    resources:
      - 
        name: {{ .VeleroMysqlHookResourceName }}
        includedNamespaces:
          - keycloak
        labelSelector:
          matchLabels:
            app: mysql
        pre:
          - 
            exec:
              container: mysql
              command:
                - bash
                - /etc/mysql/conf.d/mysql-hook.sh
                - -o backup
                - -f {{ .VeleroMysqlBackupName }}.sql
              onError: Fail
              timeout: 5m`

const VeleroMySQLRestore = `
---
apiVersion: velero.io/v1
kind: Restore
metadata:
  name: {{ .VeleroMysqlRestore }}
  namespace: {{ .VeleroNamespaceName }}
spec:
  backupName: {{ .VeleroMysqlBackupName }}
  includedNamespaces:
    - keycloak 
  restorePVs: false
  hooks:
    resources:
      - name: {{ .VeleroMysqlHookResourceName }}
        includedNamespaces:
          - keycloak
        labelSelector:
          matchLabels:
            app: mysql
        postHooks:
          - exec:
              container: mysql
              command:
                - bash
                - /etc/mysql/conf.d/mysql-hook.sh
                - -o restore
                - -f {{ .VeleroMysqlBackupName }}.sql
              waitTimeout: 5m
              execTimeout: 5m
              onError: Fail`

const InnoDBBackupS3 = `
---
apiVersion: mysql.oracle.com/v2
kind: MySQLBackup
metadata: 
  name: {{ .InnoDBBackupName }}
  namespace: {{ .InnoDBNamespaceName }}
spec: 
  clusterName: {{ .InnoDBClusterName }}
  backupProfile: 
      name: {{ .InnoDBBackupProfileName}}
      dumpInstance: 
        storage: 
          s3: 
            bucketName: {{ .InnoDBBackupObjectStoreBucketName }}
            config: {{ .InnoDBBackupCredentialsName }}
            endpoint: "https://{{ .InnoDBObjectStorageNamespaceName }}.compat.objectstorage.{{ .InnoDBBackupRegion }}.oraclecloud.com"
            prefix: {{ .InnoDBBackupStorageName }}
            profile: default`

const InnoDBBackupOci = `
---
apiVersion: mysql.oracle.com/v2
kind: MySQLBackup
metadata: 
  name: {{ .InnoDBBackupName }}
  namespace: {{ .InnoDBNamespaceName }}
spec: 
  clusterName: {{ .InnoDBClusterName }}
  backupProfile: 
      name: {{ .InnoDBBackupProfileName}}
      dumpInstance: 
        storage:
          ociObjectStorage:
            prefix: {{ .InnoDBBackupStorageName }}
            bucketName: {{ .InnoDBBackupObjectStoreBucketName }}
            credentials: {{ .InnoDBBackupCredentialsName }}`
