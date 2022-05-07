// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package types

import (
	"time"
)

// Connection data object used to communicate with Object Store
type ConnectionData struct {
	Secret     ObjectStoreSecret `json:"secret"`
	Endpoint   string            `json:"endpoint"`
	RegionName string            `json:"region_name"`
	BucketName string            `json:"bucket_name"`
	BackupName string            `json:"backup_name"`
}

//ObjectStoreSecret to render secret details
type ObjectStoreSecret struct {
	SecretName      string `json:"secret_name"`
	SecretKey       string `json:"secret_key"`
	ObjectAccessKey string `json:"object_store_access_key"`
	ObjectSecretKey string `json:"object_store_secret_key"`
}

//VeleroBackupStorageLocation defines the spec for BSL
type VeleroBackupStorageLocation struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
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
		BackupSyncPeriod string `json:"backupSyncPeriod"`
		Config           struct {
			Region           string `json:"region"`
			S3ForcePathStyle string `json:"s3ForcePathStyle"`
			S3URL            string `json:"s3Url"`
		} `json:"config"`
		Credential struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"credential"`
		ObjectStorage struct {
			Bucket string `json:"bucket"`
			Prefix string `json:"prefix"`
		} `json:"objectStorage"`
		Provider string `json:"provider"`
	} `json:"spec"`
	Status struct {
		LastSyncedTime     time.Time `json:"lastSyncedTime"`
		LastValidationTime time.Time `json:"lastValidationTime"`
		Phase              string    `json:"phase"`
	} `json:"status"`
}

//VeleroBackup defines the spec for backup
type VeleroBackup struct {
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
				Pre  []struct {
					Exec struct {
						Command   []string `json:"command"`
						Container string   `json:"container"`
						OnError   string   `json:"onError"`
						Timeout   string   `json:"timeout"`
					} `json:"exec"`
				} `json:"pre"`
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

//OpenSearchHealthResponse used to determine health details
type OpenSearchHealthResponse struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	DiscoveredMaster            bool    `json:"discovered_master"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

//OpenSearchSnapshotRequestPayload struct for registering a snapshot
type OpenSearchSnapshotRequestPayload struct {
	Type     string `json:"type"`
	Settings struct {
		Client          string `json:"client"`
		Bucket          string `json:"bucket"`
		Region          string `json:"region"`
		Endpoint        string `json:"endpoint"`
		PathStyleAccess bool   `json:"path_style_access"`
	} `json:"settings"`
}

//OpenSearchOperationResponse to render common operational responses
type OpenSearchOperationResponse struct {
	Acknowledged bool `json:"acknowledged,omitempty"`
}

//OpenSearchSnapshotResponse to render snapshot response
type OpenSearchSnapshotResponse struct {
	Accepted bool `json:"accepted,omitempty"`
}

//OpenSearchSnapshotStatus to render snapshot status
type OpenSearchSnapshotStatus struct {
	Snapshots []struct {
		Snapshot           string        `json:"snapshot"`
		UUID               string        `json:"uuid"`
		VersionID          int           `json:"version_id"`
		Version            string        `json:"version"`
		Indices            []string      `json:"indices"`
		DataStreams        []string      `json:"data_streams"`
		IncludeGlobalState bool          `json:"include_global_state"`
		State              string        `json:"state"`
		StartTime          time.Time     `json:"start_time"`
		StartTimeInMillis  int64         `json:"start_time_in_millis"`
		EndTime            time.Time     `json:"end_time"`
		EndTimeInMillis    int64         `json:"end_time_in_millis"`
		DurationInMillis   int           `json:"duration_in_millis"`
		Failures           []interface{} `json:"failures"`
		Shards             struct {
			Total      int `json:"total"`
			Failed     int `json:"failed"`
			Successful int `json:"successful"`
		} `json:"shards"`
	} `json:"snapshots"`
}

//OpenSearchDataStreams struct to render datat streams info
type OpenSearchDataStreams struct {
	DataStreams []struct {
		Name           string `json:"name"`
		TimestampField struct {
			Name string `json:"name"`
		} `json:"timestamp_field"`
		Indices []struct {
			IndexName string `json:"index_name"`
			IndexUUID string `json:"index_uuid"`
		} `json:"indices"`
		Generation int    `json:"generation"`
		Status     string `json:"status"`
		Template   string `json:"template"`
	} `json:"data_streams"`
}

//OpenSearchClusterInfo renders opensearch cluster reachability
type OpenSearchClusterInfo struct {
	Name        string `json:"name"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	Version     struct {
		Distribution                     string    `json:"distribution"`
		Number                           string    `json:"number"`
		BuildType                        string    `json:"build_type"`
		BuildHash                        string    `json:"build_hash"`
		BuildDate                        time.Time `json:"build_date"`
		BuildSnapshot                    bool      `json:"build_snapshot"`
		LuceneVersion                    string    `json:"lucene_version"`
		MinimumWireCompatibilityVersion  string    `json:"minimum_wire_compatibility_version"`
		MinimumIndexCompatibilityVersion string    `json:"minimum_index_compatibility_version"`
	} `json:"version"`
	Tagline string `json:"tagline"`
}

//OpenSearchSecureSettingsReloadStatus renders status of nodes on reload secure settings
type OpenSearchSecureSettingsReloadStatus struct {
	ClusterNodes struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_nodes"`
	ClusterName string `json:"cluster_name"`
	Nodes       struct {
		IDLGk3ZOSE5LremeiFu8Q struct {
			Name string `json:"name"`
		} `json:"IdLGk3ZOSE-5LremeiFu8Q"`
		ZeroT3P0FguQHy5X8GoUaDr5A struct {
			Name string `json:"name"`
		} `json:"0t3P0fguQHy5X8GoUaDr5A"`
		T5M9GyIhRjGrovtZtfKSQ struct {
			Name string `json:"name"`
		} `json:"t5m9gyIhRjGrovt-ztfKSQ"`
		ThreeORVTwsGScWZ4OKwRyACg struct {
			Name string `json:"name"`
		} `json:"3ORVTwsGScWZ4oKwRy_aCg"`
		FDCaHPxYTLCPiXrFtF0LeA struct {
			Name string `json:"name"`
		} `json:"FDCaHPxYTLCPiXrFtF0LeA"`
		GilrPCNeTMuxpqxjXjQEqw struct {
			Name string `json:"name"`
		} `json:"gilrPCNeTMuxpqxjXjQEqw"`
		QElkkbsRuqe2LXIDqeIBg struct {
			Name string `json:"name"`
		} `json:"qElkkbs-Ruqe2LXIDqeIBg"`
	} `json:"nodes"`
}
