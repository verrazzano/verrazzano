// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
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

// Used for filling the body of the API request to create the OCNE cluster
type RancherOCNECluster struct {
	DockerRootDir           string               `json:"dockerRootDir"`
	EnableClusterAlerting   bool                 `json:"enableClusterAlerting"`
	EnableClusterMonitoring bool                 `json:"enableClusterMonitoring"`
	EnableNetworkPolicy     bool                 `json:"enableNetworkPolicy"`
	WindowsPreferedCluster  bool                 `json:"windowsPreferedCluster"`
	Type                    string               `json:"type"`
	Name                    string               `json:"name"`
	OciocneEngineConfig     RancherOCIOCNEEngine `json:"ociocneEngineConfig"`
	CloudCredentialID       string               `json:"cloudCredentialId"`
	Labels                  struct {
	} `json:"labels"`
}
type RancherOCIOCNEEngine struct {
	CloudCredentialID     string   `json:"cloudCredentialId"`
	ClusterCidr           string   `json:"clusterCidr"`
	CompartmentID         string   `json:"compartmentId"`
	ControlPlaneMemoryGbs int      `json:"controlPlaneMemoryGbs"`
	ControlPlaneOcpus     int      `json:"controlPlaneOcpus"`
	ControlPlaneShape     string   `json:"controlPlaneShape"`
	ControlPlaneSubnet    string   `json:"controlPlaneSubnet"`
	ControlPlaneVolumeGbs int      `json:"controlPlaneVolumeGbs"`
	CorednsImageTag       string   `json:"corednsImageTag"`
	DisplayName           string   `json:"displayName"`
	DriverName            string   `json:"driverName"`
	EtcdImageTag          string   `json:"etcdImageTag"`
	ImageDisplayName      string   `json:"imageDisplayName"`
	ImageID               string   `json:"imageId"`
	InstallCalico         bool     `json:"installCalico"`
	InstallCcm            bool     `json:"installCcm"`
	InstallVerrazzano     bool     `json:"installVerrazzano"`
	KubernetesVersion     string   `json:"kubernetesVersion"`
	LoadBalancerSubnet    string   `json:"loadBalancerSubnet"`
	Name                  string   `json:"name"`
	NodePublicKeyContents string   `json:"nodePublicKeyContents"`
	NumControlPlaneNodes  int      `json:"numControlPlaneNodes"`
	OcneVersion           string   `json:"ocneVersion"`
	PodCidr               string   `json:"podCidr"`
	PrivateRegistry       string   `json:"privateRegistry"`
	ProxyEndpoint         string   `json:"proxyEndpoint"`
	Region                string   `json:"region"`
	SkipOcneInstall       bool     `json:"skipOcneInstall"`
	TigeraImageTag        string   `json:"tigeraImageTag"`
	UseNodePvEncryption   bool     `json:"useNodePvEncryption"`
	VcnID                 string   `json:"vcnId"`
	VerrazzanoResource    string   `json:"verrazzanoResource"`
	VerrazzanoTag         string   `json:"verrazzanoTag"`
	VerrazzanoVersion     string   `json:"verrazzanoVersion"`
	WorkerNodeSubnet      string   `json:"workerNodeSubnet"`
	Type                  string   `json:"type"`
	ClusterName           string   `json:"clusterName"`
	NodeShape             string   `json:"nodeShape"`
	NumWorkerNodes        int      `json:"numWorkerNodes"`
	NodePools             []string `json:"nodePools"`
	ApplyYamls            []string `json:"applyYamls"`
}

// Fills in all the values of this RancherOCNECluster object according to the values taken from environment variables
func (roc *RancherOCNECluster) fillValues(clusterName, nodePublicKeyContents, credentialID string, nodePools []string) {
	roc.OciocneEngineConfig.ClusterCidr = clusterCidr
	roc.OciocneEngineConfig.ControlPlaneMemoryGbs = controlPlaneMemoryGbs
	roc.OciocneEngineConfig.ControlPlaneOcpus = controlPlaneOcpus
	roc.OciocneEngineConfig.ControlPlaneShape = controlPlaneShape
	roc.OciocneEngineConfig.ControlPlaneVolumeGbs = controlPlaneVolumeGbs
	roc.OciocneEngineConfig.CorednsImageTag = corednsImageTag
	roc.OciocneEngineConfig.DriverName = "ociocneengine"
	roc.OciocneEngineConfig.EtcdImageTag = etcdImageTag
	roc.OciocneEngineConfig.ImageDisplayName = imageDisplayName
	roc.OciocneEngineConfig.ImageID = imageID
	roc.OciocneEngineConfig.InstallCalico = installCalico
	roc.OciocneEngineConfig.InstallCcm = installCcm
	roc.OciocneEngineConfig.InstallVerrazzano = installVerrazzano
	roc.OciocneEngineConfig.KubernetesVersion = kubernetesVersion
	roc.OciocneEngineConfig.Name = ""
	roc.OciocneEngineConfig.NumControlPlaneNodes = numControlPlaneNodes
	roc.OciocneEngineConfig.OcneVersion = ocneVersion
	roc.OciocneEngineConfig.PodCidr = podCidr
	roc.OciocneEngineConfig.PrivateRegistry = privateRegistry
	roc.OciocneEngineConfig.ProxyEndpoint = proxyEndpoint
	roc.OciocneEngineConfig.Region = region
	roc.OciocneEngineConfig.SkipOcneInstall = skipOcneInstall
	roc.OciocneEngineConfig.TigeraImageTag = tigeraImageTag
	roc.OciocneEngineConfig.UseNodePvEncryption = useNodePvEncryption
	roc.OciocneEngineConfig.VerrazzanoResource = verrazzanoResource
	roc.OciocneEngineConfig.VerrazzanoTag = verrazzanoTag
	roc.OciocneEngineConfig.VerrazzanoVersion = verrazzanoVersion
	roc.OciocneEngineConfig.Type = "ociocneEngineConfig"
	roc.OciocneEngineConfig.ClusterName = ""
	roc.OciocneEngineConfig.NodeShape = nodeShape
	roc.OciocneEngineConfig.NumWorkerNodes = numWorkerNodes
	roc.OciocneEngineConfig.CloudCredentialID = credentialID
	roc.OciocneEngineConfig.CompartmentID = compartmentID
	roc.OciocneEngineConfig.ControlPlaneSubnet = controlPlaneSubnet
	roc.OciocneEngineConfig.DisplayName = clusterName
	roc.OciocneEngineConfig.LoadBalancerSubnet = loadBalancerSubnet
	roc.OciocneEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	roc.OciocneEngineConfig.Region = region
	roc.OciocneEngineConfig.VcnID = vcnID
	roc.OciocneEngineConfig.WorkerNodeSubnet = workerNodeSubnet
	roc.OciocneEngineConfig.NodePools = nodePools
	if applyYAMLs == "" {
		roc.OciocneEngineConfig.ApplyYamls = []string{}
	} else {
		roc.OciocneEngineConfig.ApplyYamls = []string{applyYAMLs}
	}

	roc.Name = clusterName
	roc.CloudCredentialID = credentialID
	roc.DockerRootDir = dockerRootDir
	roc.EnableClusterAlerting = enableClusterAlerting
	roc.EnableClusterMonitoring = enableClusterMonitoring
	roc.EnableNetworkPolicy = enableNetworkPolicy
	roc.WindowsPreferedCluster = windowsPreferedCluster
	roc.Type = "cluster"
	roc.Labels = struct{}{}
}

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
