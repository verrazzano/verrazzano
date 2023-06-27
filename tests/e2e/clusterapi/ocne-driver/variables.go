// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"os"

	"github.com/hashicorp/go-retryablehttp"
)

var (
	// Initialized by ensureOCNEDriverVarsInitialized
	region                string
	vcnID                 string
	userID                string
	tenancyID             string
	fingerprint           string
	privateKeyPath        string
	nodePublicKeyPath     string
	compartmentID         string
	workerNodeSubnet      string
	controlPlaneSubnet    string
	loadBalancerSubnet    string
	ocneClusterNameSuffix string

	// Initialized during before suite, and used across helper functions
	rancherURL        string
	httpClient        *retryablehttp.Client
	cloudCredentialID string
)

// Grabs info from environment variables required by this test suite
func ensureOCNEDriverVarsInitialized() {
	region = os.Getenv("OCI_REGION")
	vcnID = os.Getenv("OCI_VCN_ID")
	userID = os.Getenv("OCI_USER_ID")
	tenancyID = os.Getenv("OCI_TENANCY_ID")
	fingerprint = os.Getenv("OCI_CREDENTIALS_FINGERPRINT")
	privateKeyPath = os.Getenv("OCI_PRIVATE_KEY_PATH")
	nodePublicKeyPath = os.Getenv("NODE_PUBLIC_KEY_PATH")
	compartmentID = os.Getenv("OCI_COMPARTMENT_ID")
	workerNodeSubnet = os.Getenv("WORKER_NODE_SUBNET")
	controlPlaneSubnet = os.Getenv("CONTROL_PLANE_SUBNET")
	loadBalancerSubnet = os.Getenv("LOAD_BALANCER_SUBNET")
	ocneClusterNameSuffix = os.Getenv("OCNE_CLUSTER_NAME_SUFFIX")
}

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
	CalicoImagePath       string        `json:"calicoImagePath"`
	CloudCredentialID     string        `json:"cloudCredentialId"`
	ClusterCidr           string        `json:"clusterCidr"`
	CompartmentID         string        `json:"compartmentId"`
	ControlPlaneMemoryGbs int           `json:"controlPlaneMemoryGbs"`
	ControlPlaneOcpus     int           `json:"controlPlaneOcpus"`
	ControlPlaneShape     string        `json:"controlPlaneShape"`
	ControlPlaneSubnet    string        `json:"controlPlaneSubnet"`
	ControlPlaneVolumeGbs int           `json:"controlPlaneVolumeGbs"`
	CorednsImageTag       string        `json:"corednsImageTag"`
	DisplayName           string        `json:"displayName"`
	DriverName            string        `json:"driverName"`
	EtcdImageTag          string        `json:"etcdImageTag"`
	ImageDisplayName      string        `json:"imageDisplayName"`
	ImageID               string        `json:"imageId"`
	InstallCalico         bool          `json:"installCalico"`
	InstallCcm            bool          `json:"installCcm"`
	InstallVerrazzano     bool          `json:"installVerrazzano"`
	KubernetesVersion     string        `json:"kubernetesVersion"`
	LoadBalancerSubnet    string        `json:"loadBalancerSubnet"`
	Name                  string        `json:"name"`
	NodePublicKeyContents string        `json:"nodePublicKeyContents"`
	NumControlPlaneNodes  int           `json:"numControlPlaneNodes"`
	OcneVersion           string        `json:"ocneVersion"`
	PodCidr               string        `json:"podCidr"`
	PrivateRegistry       string        `json:"privateRegistry"`
	ProxyEndpoint         string        `json:"proxyEndpoint"`
	Region                string        `json:"region"`
	SkipOcneInstall       bool          `json:"skipOcneInstall"`
	TigeraImageTag        string        `json:"tigeraImageTag"`
	UseNodePvEncryption   bool          `json:"useNodePvEncryption"`
	VcnID                 string        `json:"vcnId"`
	VerrazzanoResource    string        `json:"verrazzanoResource"`
	VerrazzanoTag         string        `json:"verrazzanoTag"`
	VerrazzanoVersion     string        `json:"verrazzanoVersion"`
	WorkerNodeSubnet      string        `json:"workerNodeSubnet"`
	Type                  string        `json:"type"`
	ClusterName           string        `json:"clusterName"`
	NodeShape             string        `json:"nodeShape"`
	NumWorkerNodes        int           `json:"numWorkerNodes"`
	NodePools             []interface{} `json:"nodePools"`
	ApplyYamls            []interface{} `json:"applyYamls"`
}

// Fills in the common constant values used across all the cluster creation scenarios
func (roc *RancherOCNECluster) fillConstantValues() {
	roc.OciocneEngineConfig.CalicoImagePath = "olcne"
	roc.OciocneEngineConfig.ClusterCidr = "10.96.0.0/16"
	roc.OciocneEngineConfig.ControlPlaneMemoryGbs = 16
	roc.OciocneEngineConfig.ControlPlaneOcpus = 2
	roc.OciocneEngineConfig.ControlPlaneShape = "VM.Standard.E4.Flex"
	roc.OciocneEngineConfig.ControlPlaneVolumeGbs = 100
	roc.OciocneEngineConfig.CorednsImageTag = "v1.9.3"
	roc.OciocneEngineConfig.DriverName = "ociocneengine"
	roc.OciocneEngineConfig.EtcdImageTag = "3.5.6"
	roc.OciocneEngineConfig.ImageDisplayName = "Oracle-Linux-8.7-2023.05.24-0"
	roc.OciocneEngineConfig.ImageID = ""
	roc.OciocneEngineConfig.InstallCalico = true
	roc.OciocneEngineConfig.InstallCcm = true
	roc.OciocneEngineConfig.InstallVerrazzano = false
	roc.OciocneEngineConfig.KubernetesVersion = "v1.25.7"
	roc.OciocneEngineConfig.Name = ""
	roc.OciocneEngineConfig.NumControlPlaneNodes = 1
	roc.OciocneEngineConfig.OcneVersion = "1.6"
	roc.OciocneEngineConfig.PodCidr = "10.244.0.0/16"
	roc.OciocneEngineConfig.PrivateRegistry = ""
	roc.OciocneEngineConfig.ProxyEndpoint = ""
	roc.OciocneEngineConfig.Region = region
	roc.OciocneEngineConfig.SkipOcneInstall = false
	roc.OciocneEngineConfig.TigeraImageTag = "v1.29.0"
	roc.OciocneEngineConfig.UseNodePvEncryption = true
	roc.OciocneEngineConfig.VerrazzanoResource = "apiVersion: install.verrazzano.io/v1beta1\nkind: Verrazzano\nmetadata:\n  name: managed\n  namespace: default\nspec:\n  profile: managed-cluster"
	roc.OciocneEngineConfig.VerrazzanoTag = "v1.6.0-20230609132620-44e8f4d1"
	roc.OciocneEngineConfig.VerrazzanoVersion = "1.6.0-4574+44e8f4d1"
	roc.OciocneEngineConfig.Type = "ociocneEngineConfig"
	roc.OciocneEngineConfig.ClusterName = ""
	roc.OciocneEngineConfig.NodeShape = "VM.Standard.E4.Flex"
	roc.OciocneEngineConfig.NumWorkerNodes = 1
	roc.OciocneEngineConfig.ApplyYamls = []interface{}{}

	roc.DockerRootDir = "/var/lib/docker"
	roc.EnableClusterAlerting = false
	roc.EnableClusterMonitoring = false
	roc.EnableNetworkPolicy = false
	roc.WindowsPreferedCluster = false
	roc.Type = "cluster"
	roc.Labels = struct{}{}
}
