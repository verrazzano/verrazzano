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

// VeleroBackupModel defines the spec for backup
type VeleroBackupModel struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KubectlKubernetesIoLastAppliedConfiguration string `json:"kubectl.kubernetes.io/last-applied-configuration"`
			VeleroIoSourceClusterK8SGitversion          string `json:"velero.io/source-cluster-k8s-gitversion"`
			VeleroIoSourceClusterK8SMajorVersion        string `json:"velero.io/source-cluster-k8s-major-version"`
			VeleroIoSourceClusterK8SMinorVersion        string `json:"velero.io/source-cluster-k8s-minor-version"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Generation        int       `json:"generation"`
		Labels            struct {
			VeleroIoStorageLocation string `json:"velero.io/storage-location"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		DefaultVolumesToRestic bool `json:"defaultVolumesToRestic"`
		Hooks                  struct {
			Resources []struct {
				IncludedNamespaces []string `json:"includedNamespaces"`
				LabelSelector      struct {
					MatchLabels struct {
						App string `json:"app"`
					} `json:"matchLabels"`
				} `json:"labelSelector"`
				Name string `json:"name"`
				Post []struct {
					Exec struct {
						Command   []string `json:"command"`
						Container string   `json:"container"`
						OnError   string   `json:"onError"`
						Timeout   string   `json:"timeout"`
					} `json:"exec"`
				} `json:"post"`
			} `json:"resources"`
		} `json:"hooks"`
		IncludedNamespaces []string `json:"includedNamespaces"`
		StorageLocation    string   `json:"storageLocation"`
		TTL                string   `json:"ttl"`
	} `json:"spec"`
	Status struct {
		CompletionTimestamp time.Time `json:"completionTimestamp"`
		Expiration          time.Time `json:"expiration"`
		FormatVersion       string    `json:"formatVersion"`
		Phase               string    `json:"phase"`
		Progress            struct {
			ItemsBackedUp int `json:"itemsBackedUp"`
			TotalItems    int `json:"totalItems"`
		} `json:"progress"`
		StartTimestamp time.Time `json:"startTimestamp"`
		Version        int       `json:"version"`
	} `json:"status"`
}

type VeleroRestoreModel struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
	Spec struct {
		BackupName         string   `json:"backupName"`
		IncludedNamespaces []string `json:"includedNamespaces"`
		ExcludedNamespaces []string `json:"excludedNamespaces"`
		IncludedResources  []string `json:"includedResources"`
		ExcludedResources  []string `json:"excludedResources"`
		RestoreStatus      struct {
			IncludedResources []string      `json:"includedResources"`
			ExcludedResources []interface{} `json:"excludedResources"`
		} `json:"restoreStatus"`
		IncludeClusterResources interface{} `json:"includeClusterResources"`
		LabelSelector           struct {
			MatchLabels struct {
				App       string `json:"app"`
				Component string `json:"component"`
			} `json:"matchLabels"`
		} `json:"labelSelector"`
		OrLabelSelectors []struct {
			MatchLabels struct {
				App string `json:"app"`
			} `json:"matchLabels"`
		} `json:"orLabelSelectors"`
		NamespaceMapping struct {
			NamespaceBackupFrom string `json:"namespace-backup-from"`
		} `json:"namespaceMapping"`
		RestorePVs             bool   `json:"restorePVs"`
		ScheduleName           string `json:"scheduleName"`
		ExistingResourcePolicy string `json:"existingResourcePolicy"`
		Hooks                  struct {
			Resources []struct {
				Name               string        `json:"name"`
				IncludedNamespaces []string      `json:"includedNamespaces"`
				ExcludedNamespaces []string      `json:"excludedNamespaces"`
				IncludedResources  []string      `json:"includedResources"`
				ExcludedResources  []interface{} `json:"excludedResources"`
				LabelSelector      struct {
					MatchLabels struct {
						App       string `json:"app"`
						Component string `json:"component"`
					} `json:"matchLabels"`
				} `json:"labelSelector"`
				PostHooks []struct {
					Init struct {
						InitContainers []struct {
							Name         string `json:"name"`
							Image        string `json:"image"`
							VolumeMounts []struct {
								MountPath string `json:"mountPath"`
								Name      string `json:"name"`
							} `json:"volumeMounts"`
							Command []string `json:"command"`
						} `json:"initContainers"`
					} `json:"init,omitempty"`
					Exec struct {
						Container   string   `json:"container"`
						Command     []string `json:"command"`
						WaitTimeout string   `json:"waitTimeout"`
						ExecTimeout string   `json:"execTimeout"`
						OnError     string   `json:"onError"`
					} `json:"exec,omitempty"`
				} `json:"postHooks"`
			} `json:"resources"`
		} `json:"hooks"`
	} `json:"spec"`
	Status struct {
		Phase            string      `json:"phase"`
		ValidationErrors interface{} `json:"validationErrors"`
		Warnings         int         `json:"warnings"`
		Errors           int         `json:"errors"`
		FailureReason    interface{} `json:"failureReason"`
	} `json:"status"`
}

type RancherBackupModel struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KubectlKubernetesIoLastAppliedConfiguration string `json:"kubectl.kubernetes.io/last-applied-configuration"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Generation        int       `json:"generation"`
		Name              string    `json:"name"`
		ResourceVersion   string    `json:"resourceVersion"`
		UID               string    `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		ResourceSetName string `json:"resourceSetName"`
		StorageLocation struct {
			S3 struct {
				BucketName                string `json:"bucketName"`
				CredentialSecretName      string `json:"credentialSecretName"`
				CredentialSecretNamespace string `json:"credentialSecretNamespace"`
				Endpoint                  string `json:"endpoint"`
				Folder                    string `json:"folder"`
				Region                    string `json:"region"`
			} `json:"s3"`
		} `json:"storageLocation"`
	} `json:"spec"`
	Status struct {
		BackupType string `json:"backupType"`
		Conditions []struct {
			LastUpdateTime time.Time `json:"lastUpdateTime"`
			Message        string    `json:"message,omitempty"`
			Status         string    `json:"status"`
			Type           string    `json:"type"`
		} `json:"conditions"`
		Filename           string    `json:"filename"`
		LastSnapshotTs     time.Time `json:"lastSnapshotTs"`
		NextSnapshotAt     string    `json:"nextSnapshotAt"`
		ObservedGeneration int       `json:"observedGeneration"`
		StorageLocation    string    `json:"storageLocation"`
		Summary            string    `json:"summary"`
	} `json:"status"`
}

type RancherRestoreModel struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KubectlKubernetesIoLastAppliedConfiguration string `json:"kubectl.kubernetes.io/last-applied-configuration"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Generation        int       `json:"generation"`
		Name              string    `json:"name"`
		ResourceVersion   string    `json:"resourceVersion"`
		UID               string    `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		BackupFilename  string `json:"backupFilename"`
		StorageLocation struct {
			S3 struct {
				BucketName                string `json:"bucketName"`
				CredentialSecretName      string `json:"credentialSecretName"`
				CredentialSecretNamespace string `json:"credentialSecretNamespace"`
				Endpoint                  string `json:"endpoint"`
				Folder                    string `json:"folder"`
				Region                    string `json:"region"`
			} `json:"s3"`
		} `json:"storageLocation"`
	} `json:"spec"`
	Status struct {
		BackupSource string `json:"backupSource"`
		Conditions   []struct {
			LastUpdateTime time.Time `json:"lastUpdateTime"`
			Message        string    `json:"message"`
			Status         string    `json:"status"`
			Type           string    `json:"type"`
		} `json:"conditions"`
		ObservedGeneration  int       `json:"observedGeneration"`
		RestoreCompletionTs time.Time `json:"restoreCompletionTs"`
		Summary             string    `json:"summary"`
	} `json:"status"`
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
	RancherUserIDList           []string
	RancherUserNameList         []string
	KeyCloakUserIDList          []string
)
