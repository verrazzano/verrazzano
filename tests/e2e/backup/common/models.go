// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"bytes"
	"time"
)

// RancherBackupData struct used for rancher backup templating
type RancherBackupData struct {
	RancherBackupName string
	RancherSecretData RancherObjectStoreData
}

// RancherRestoreData struct used for rancher restore templating
type RancherRestoreData struct {
	RancherRestoreName string
	BackupFileName     string
	RancherSecretData  RancherObjectStoreData
}

// RancherObjectStoreData struct used for rancher secret templating
type RancherObjectStoreData struct {
	RancherSecretName                 string
	RancherSecretNamespaceName        string
	RancherObjectStoreBucketName      string
	RancherBackupRegion               string
	RancherObjectStorageNamespaceName string
}

// BashCommand struct used for running bash commands
type BashCommand struct {
	Timeout     time.Duration `json:"timeout"`
	CommandArgs []string      `json:"cmdArgs"`
}

// RunnerResponse is structured response for bash commands
type RunnerResponse struct {
	StandardOut  bytes.Buffer `json:"stdout"`
	StandardErr  bytes.Buffer `json:"stderr"`
	CommandError error        `json:"error"`
}

// AccessData struct used for velero secrets templating
type AccessData struct {
	AccessName             string
	ScrtName               string
	ObjectStoreAccessValue string
	ObjectStoreScrt        string
}

// VeleroBackupLocationObjectData holds data related to velero backup location
type VeleroBackupLocationObjectData struct {
	VeleroBackupStorageName          string
	VeleroNamespaceName              string
	VeleroObjectStoreBucketName      string
	VeleroSecretName                 string
	VeleroObjectStorageNamespaceName string
	VeleroBackupRegion               string
}

// VeleroBackupObject holds data related to velero backup
type VeleroBackupObject struct {
	VeleroBackupName                 string
	VeleroNamespaceName              string
	VeleroBackupStorageName          string
	VeleroOpensearchHookResourceName string
}

// VeleroRestoreObject holds data related to velero restore
type VeleroRestoreObject struct {
	VeleroRestore                    string
	VeleroNamespaceName              string
	VeleroBackupName                 string
	VeleroOpensearchHookResourceName string
}

// EsQueryObject holds data related to opensearch index query
type EsQueryObject struct {
	BackupIDBeforeBackup string
}

// RancherUser holds data related to rancher test user
type RancherUser struct {
	FullName string
	Password string
	Username string
}

type VeleroMysqlBackupObject struct {
	VeleroMysqlBackupName        string
	VeleroNamespaceName          string
	VeleroMysqlBackupStorageName string
	VeleroMysqlHookResourceName  string
}

type VeleroMysqlRestoreObject struct {
	VeleroMysqlRestore          string
	VeleroNamespaceName         string
	VeleroMysqlBackupName       string
	VeleroMysqlHookResourceName string
}

// Variables used across backup components
var (
	VeleroNameSpace             string
	VeleroOpenSearchSecretName  string
	VeleroMySQLSecretName       string
	RancherSecretName           string
	OciBucketID                 string
	OciBucketName               string
	OciOsAccessKey              string
	OciOsAccessSecretKey        string
	OciCompartmentID            string
	OciNamespaceName            string
	BackupResourceName          string
	BackupOpensearchName        string
	BackupRancherName           string
	BackupMySQLName             string
	RestoreOpensearchName       string
	RestoreRancherName          string
	RestoreMySQLName            string
	BackupRegion                string
	BackupOpensearchStorageName string
	BackupMySQLStorageName      string
	BackupID                    string
	RancherURL                  string
	RancherBackupFileName       string
	RancherToken                string
	KeyCloakUserIDList          []string
)
