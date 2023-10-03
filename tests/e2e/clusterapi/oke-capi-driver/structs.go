// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package okecapidriver

import (
	"github.com/Masterminds/semver/v3"
	"time"
)

// Used for filling the body of the API request to create the cloud credential
type RancherCloudCred struct {
	Type     string `json:"type"`
	Metadata struct {
		GenerateName string `json:"generateName"`
		Namespace    string `json:"namespace"`
	} `json:"metadata"`
	InternalName string `json:"_name"`
	Annotations  struct {
		ProvisioningCattleIoDriver string `json:"provisioning.cattle.io/driver"`
	} `json:"annotations"`
	RancherOcicredentialConfig `json:"ocicredentialConfig"`
	InternalType               string `json:"_type"`
	Name                       string `json:"name"`
}
type RancherOcicredentialConfig struct {
	Fingerprint        string `json:"fingerprint"`
	PrivateKeyContents string `json:"privateKeyContents"`
	Region             string `json:"region"`
	TenancyID          string `json:"tenancyId"`
	UserID             string `json:"userId"`
}

// Used for filling the body of the API request to create/update the OKE cluster
type RancherOKECluster struct {
	DockerRootDir           string               `json:"dockerRootDir"`
	EnableClusterAlerting   bool                 `json:"enableClusterAlerting"`
	EnableClusterMonitoring bool                 `json:"enableClusterMonitoring"`
	EnableNetworkPolicy     bool                 `json:"enableNetworkPolicy"`
	WindowsPreferedCluster  bool                 `json:"windowsPreferedCluster"`
	Type                    string               `json:"type"`
	Name                    string               `json:"name"`
	Description             string               `json:"description"`
	OKECAPIEngineConfig     RancherOKECAPIEngine `json:"okecapiEngineConfig"`
	CloudCredentialID       string               `json:"cloudCredentialId"`
	Labels                  struct {
	} `json:"labels"`
}
type RancherOKECAPIEngine struct {
	CloudCredentialID     string   `json:"cloudCredentialId"`
	ClusterCidr           string   `json:"clusterCidr"`
	CompartmentID         string   `json:"compartmentId"`
	ControlPlaneSubnet    string   `json:"controlPlaneSubnet"`
	DisplayName           string   `json:"displayName"`
	DriverName            string   `json:"driverName"`
	ImageDisplayName      string   `json:"imageDisplayName"`
	ImageID               string   `json:"imageId"`
	KubernetesVersion     string   `json:"kubernetesVersion"`
	LoadBalancerSubnet    string   `json:"loadBalancerSubnet"`
	Name                  string   `json:"name"`
	NodePublicKeyContents string   `json:"nodePublicKeyContents"`
	PodCidr               string   `json:"podCidr"`
	Region                string   `json:"region"`
	VcnID                 string   `json:"vcnId"`
	WorkerNodeSubnet      string   `json:"workerNodeSubnet"`
	Type                  string   `json:"type"`
	NodeShape             string   `json:"nodeShape"`
	NodePools             []string `json:"nodePools"`
	ApplyYamls            []string `json:"applyYamls"`
	InstallVerrazzano     bool     `json:"installVerrazzano"`
}

// Fills in values of this RancherOKECluster object according to the values taken from environment variables.
// Other fields of the struct that are not filled by environment variables are not filled in.
func (roc *RancherOKECluster) fillCommonValues() {
	roc.OKECAPIEngineConfig.ClusterCidr = clusterCidr
	roc.OKECAPIEngineConfig.DriverName = "okecapi"
	roc.OKECAPIEngineConfig.ImageDisplayName = imageDisplayName
	roc.OKECAPIEngineConfig.ImageID = imageID
	roc.OKECAPIEngineConfig.KubernetesVersion = kubernetesVersion
	roc.OKECAPIEngineConfig.Name = ""
	roc.OKECAPIEngineConfig.PodCidr = podCidr
	roc.OKECAPIEngineConfig.Region = region
	roc.OKECAPIEngineConfig.Type = "okecapiEngineConfig"
	roc.OKECAPIEngineConfig.NodeShape = nodeShape
	roc.OKECAPIEngineConfig.CompartmentID = compartmentID
	roc.OKECAPIEngineConfig.ControlPlaneSubnet = controlPlaneSubnet
	roc.OKECAPIEngineConfig.LoadBalancerSubnet = loadBalancerSubnet
	roc.OKECAPIEngineConfig.Region = region
	roc.OKECAPIEngineConfig.VcnID = vcnID
	roc.OKECAPIEngineConfig.WorkerNodeSubnet = workerNodeSubnet
	if applyYAMLs == "" {
		roc.OKECAPIEngineConfig.ApplyYamls = []string{}
	} else {
		roc.OKECAPIEngineConfig.ApplyYamls = []string{applyYAMLs}
	}
	roc.OKECAPIEngineConfig.InstallVerrazzano = installVerrazzano

	roc.DockerRootDir = dockerRootDir
	roc.EnableClusterAlerting = enableClusterAlerting
	roc.EnableClusterMonitoring = enableClusterMonitoring
	roc.EnableNetworkPolicy = enableNetworkPolicy
	roc.WindowsPreferedCluster = windowsPreferedCluster
	roc.Type = "cluster"
	roc.Labels = struct{}{}
}

// Represents the clusters.provisioning.cattle.io object
type ProvisioningCluster struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Metadata   struct {
		Annotations struct {
			FieldCattleIoCreatorID             string `json:"field.cattle.io/creatorId"`
			ObjectsetRioCattleIoApplied        string `json:"objectset.rio.cattle.io/applied"`
			ObjectsetRioCattleIoID             string `json:"objectset.rio.cattle.io/id"`
			ObjectsetRioCattleIoOwnerGvk       string `json:"objectset.rio.cattle.io/owner-gvk"`
			ObjectsetRioCattleIoOwnerName      string `json:"objectset.rio.cattle.io/owner-name"`
			ObjectsetRioCattleIoOwnerNamespace string `json:"objectset.rio.cattle.io/owner-namespace"`
		} `json:"annotations"`
		CreationTimestamp time.Time `json:"creationTimestamp"`
		Finalizers        []string  `json:"finalizers"`
		Generation        int       `json:"generation"`
		Labels            struct {
			ObjectsetRioCattleIoHash string `json:"objectset.rio.cattle.io/hash"`
		} `json:"labels"`
		Name            string `json:"name"`
		Namespace       string `json:"namespace"`
		ResourceVersion string `json:"resourceVersion"`
		UID             string `json:"uid"`
	} `json:"metadata"`
	Spec struct {
		LocalClusterAuthEndpoint struct {
		} `json:"localClusterAuthEndpoint"`
	} `json:"spec"`
	Status struct {
		AgentDeployed    bool   `json:"agentDeployed"`
		ClientSecretName string `json:"clientSecretName"`
		ClusterName      string `json:"clusterName"`
		Conditions       []struct {
			Status         string    `json:"status"`
			Type           string    `json:"type"`
			LastUpdateTime time.Time `json:"lastUpdateTime,omitempty"`
		} `json:"conditions"`
		ObservedGeneration int  `json:"observedGeneration"`
		Ready              bool `json:"ready"`
	} `json:"status"`
}

type OKEMetadataItem struct {
	KubernetesVersion *semver.Version
}
