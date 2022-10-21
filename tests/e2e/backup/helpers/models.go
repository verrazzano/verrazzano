// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

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
	CommandArgs []string `json:"cmdArgs"`
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

type InnoDBSecret struct {
	AccessName string
	ScrtName   string
	Region     string
}

type InnoDBBackupObject struct {
	InnoDBBackupName                  string
	InnoDBNamespaceName               string
	InnoDBClusterName                 string
	InnoDBBackupProfileName           string
	InnoDBBackupObjectStoreBucketName string
	InnoDBObjectStorageNamespaceName  string
	InnoDBBackupCredentialsName       string
	InnoDBBackupStorageName           string
	InnoDBBackupRegion                string
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

type VeleroPodVolumeBackups struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		CreationTimestamp time.Time `json:"creationTimestamp"`
		GenerateName      string    `json:"generateName"`
		Generation        int       `json:"generation"`
		Labels            struct {
			VeleroIoBackupName string `json:"velero.io/backup-name"`
			VeleroIoBackupUID  string `json:"velero.io/backup-uid"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		OwnerReferences []struct {
			APIVersion string `json:"apiVersion"`
			Controller bool   `json:"controller"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			UID        string `json:"uid"`
		} `json:"ownerReferences"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		BackupStorageLocation string `json:"backupStorageLocation"`
		Node                  string `json:"node"`
		Pod                   struct {
			Kind      string `json:"kind"`
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			UID       string `json:"uid"`
		} `json:"pod"`
		RepoIdentifier string `json:"repoIdentifier"`
		Tags           struct {
			Backup    string `json:"backup"`
			BackupUID string `json:"backup-uid"`
			Ns        string `json:"ns"`
			Pod       string `json:"pod"`
			PodUID    string `json:"pod-uid"`
			Volume    string `json:"volume"`
		} `json:"tags"`
		Volume string `json:"volume"`
	} `json:"spec"`
	Status struct {
		CompletionTimestamp time.Time `json:"completionTimestamp"`
		Path                string    `json:"path"`
		Phase               string    `json:"phase"`
		Progress            struct {
			BytesDone  int `json:"bytesDone"`
			TotalBytes int `json:"totalBytes"`
		} `json:"progress"`
		SnapshotID     string    `json:"snapshotID"`
		StartTimestamp time.Time `json:"startTimestamp"`
	} `json:"status"`
}

type VeleroPodVolumeRestores struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		CreationTimestamp time.Time `json:"creationTimestamp"`
		GenerateName      string    `json:"generateName"`
		Generation        int       `json:"generation"`
		Labels            struct {
			VeleroIoPodUID      string `json:"velero.io/pod-uid"`
			VeleroIoRestoreName string `json:"velero.io/restore-name"`
			VeleroIoRestoreUID  string `json:"velero.io/restore-uid"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		OwnerReferences []struct {
			APIVersion string `json:"apiVersion"`
			Controller bool   `json:"controller"`
			Kind       string `json:"kind"`
			Name       string `json:"name"`
			UID        string `json:"uid"`
		} `json:"ownerReferences"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		BackupStorageLocation string `json:"backupStorageLocation"`
		Pod                   struct {
			Kind      string `json:"kind"`
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			UID       string `json:"uid"`
		} `json:"pod"`
		RepoIdentifier string `json:"repoIdentifier"`
		SnapshotID     string `json:"snapshotID"`
		Volume         string `json:"volume"`
	} `json:"spec"`
	Status struct {
		CompletionTimestamp time.Time `json:"completionTimestamp"`
		Phase               string    `json:"phase"`
		Progress            struct {
			BytesDone  int `json:"bytesDone"`
			TotalBytes int `json:"totalBytes"`
		} `json:"progress"`
		StartTimestamp time.Time `json:"startTimestamp"`
	} `json:"status"`
}

type MySQLBackupModel struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KopfZalandoOrgLastHandledConfiguration      string `json:"kopf.zalando.org/last-handled-configuration"`
			KubectlKubernetesIoLastAppliedConfiguration string `json:"kubectl.kubernetes.io/last-applied-configuration"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Generation        int       `json:"generation"`
		Name              string    `json:"name"`
		Namespace         string    `json:"namespace"`
		ResourceVersion   string    `json:"resourceVersion"`
		UID               string    `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		AddTimestampToBackupDirectory bool `json:"addTimestampToBackupDirectory"`
		BackupProfile                 struct {
			DumpInstance struct {
				Storage struct {
					S3 struct {
						BucketName string `json:"bucketName"`
						Config     string `json:"config"`
						Endpoint   string `json:"endpoint"`
						Prefix     string `json:"prefix"`
						Profile    string `json:"profile"`
					} `json:"s3"`
				} `json:"storage"`
			} `json:"dumpInstance"`
			Name string `json:"name"`
		} `json:"backupProfile"`
		ClusterName      string `json:"clusterName"`
		DeleteBackupData bool   `json:"deleteBackupData"`
	} `json:"spec"`
	Status struct {
		Bucket         string    `json:"bucket"`
		CompletionTime time.Time `json:"completionTime"`
		ElapsedTime    string    `json:"elapsedTime"`
		Method         string    `json:"method"`
		Output         string    `json:"output"`
		Source         string    `json:"source"`
		StartTime      time.Time `json:"startTime"`
		Status         string    `json:"status"`
	} `json:"status"`
}

type InnoDBIcsModel struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			KopfZalandoOrgLastHandledConfiguration string `json:"kopf.zalando.org/last-handled-configuration"`
			MetaHelmShReleaseName                  string `json:"meta.helm.sh/release-name"`
			MetaHelmShReleaseNamespace             string `json:"meta.helm.sh/release-namespace"`
			MysqlOracleComClusterInfo              string `json:"mysql.oracle.com/cluster-info"`
			MysqlOracleComMysqlOperatorVersion     string `json:"mysql.oracle.com/mysql-operator-version"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Labels            struct {
			AppKubernetesIoManagedBy string `json:"app.kubernetes.io/managed-by"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		BaseServerID    int    `json:"baseServerId"`
		ImagePullPolicy string `json:"imagePullPolicy"`
		ImageRepository string `json:"imageRepository"`
		Instances       int    `json:"instances"`
		PodSpec         struct {
			Affinity struct {
				PodAntiAffinity struct {
					PreferredDuringSchedulingIgnoredDuringExecution []struct {
						PodAffinityTerm struct {
							LabelSelector struct {
								MatchLabels struct {
									AppKubernetesIoInstance string `json:"app.kubernetes.io/instance"`
									AppKubernetesIoName     string `json:"app.kubernetes.io/name"`
								} `json:"matchLabels"`
							} `json:"labelSelector"`
							TopologyKey string `json:"topologyKey"`
						} `json:"podAffinityTerm"`
						Weight int `json:"weight"`
					} `json:"preferredDuringSchedulingIgnoredDuringExecution"`
				} `json:"podAntiAffinity"`
			} `json:"affinity"`
			Containers []struct {
				Name         string `json:"name"`
				VolumeMounts []struct {
					MountPath string `json:"mountPath"`
					Name      string `json:"name"`
					SubPath   string `json:"subPath"`
				} `json:"volumeMounts"`
			} `json:"containers"`
			InitContainers []struct {
				Name         string `json:"name"`
				VolumeMounts []struct {
					MountPath string `json:"mountPath"`
					Name      string `json:"name"`
					SubPath   string `json:"subPath"`
				} `json:"volumeMounts"`
			} `json:"initContainers"`
			Volumes []struct {
				ConfigMap struct {
					DefaultMode int `json:"defaultMode"`
					Items       []struct {
						Key  string `json:"key"`
						Path string `json:"path"`
					} `json:"items"`
					Name string `json:"name"`
				} `json:"configMap"`
				Name string `json:"name"`
			} `json:"volumes"`
		} `json:"podSpec"`
		Router struct {
			Instances int `json:"instances"`
			PodSpec   struct {
				Affinity struct {
					PodAntiAffinity struct {
						PreferredDuringSchedulingIgnoredDuringExecution []struct {
							PodAffinityTerm struct {
								LabelSelector struct {
									MatchLabels struct {
										AppKubernetesIoInstance string `json:"app.kubernetes.io/instance"`
										AppKubernetesIoName     string `json:"app.kubernetes.io/name"`
									} `json:"matchLabels"`
								} `json:"labelSelector"`
								TopologyKey string `json:"topologyKey"`
							} `json:"podAffinityTerm"`
							Weight int `json:"weight"`
						} `json:"preferredDuringSchedulingIgnoredDuringExecution"`
					} `json:"podAntiAffinity"`
				} `json:"affinity"`
			} `json:"podSpec"`
		} `json:"router"`
		SecretName         string `json:"secretName"`
		ServiceAccountName string `json:"serviceAccountName"`
		TLSUseSelfSigned   bool   `json:"tlsUseSelfSigned"`
		Version            string `json:"version"`
	} `json:"spec"`
	Status struct {
		Cluster struct {
			LastProbeTime   time.Time `json:"lastProbeTime"`
			OnlineInstances int       `json:"onlineInstances"`
			Status          string    `json:"status"`
		} `json:"cluster"`
		CreateTime time.Time `json:"createTime"`
		Kopf       struct {
			Progress struct {
			} `json:"progress"`
		} `json:"kopf"`
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
	MySQLBackupHelmFileName     string
	OciCliTenancy               string
	OciCliUser                  string
	OciCliFingerprint           string
	OciCliKeyFile               string
	KeyCloakReplicaCount        int32
)
