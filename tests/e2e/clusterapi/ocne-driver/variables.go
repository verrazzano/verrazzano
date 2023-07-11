// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"os"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
)

var (
	// Initialized by ensureOCNEDriverVarsInitialized, required environment variables
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

	// Initialized by ensureOCNEDriverVarsInitialized, optional overrides
	dockerRootDir           string
	enableClusterAlerting   bool
	enableClusterMonitoring bool
	enableNetworkPolicy     bool
	windowsPreferedCluster  bool
	calicoImagePath         string
	clusterCidr             string
	controlPlaneMemoryGbs   int
	controlPlaneOcpus       int
	controlPlaneShape       string
	controlPlaneVolumeGbs   int
	corednsImageTag         string
	etcdImageTag            string
	imageDisplayName        string
	imageID                 string
	installCalico           bool
	installCcm              bool
	installVerrazzano       bool
	kubernetesVersion       string
	numControlPlaneNodes    int
	ocneVersion             string
	podCidr                 string
	privateRegistry         string
	proxyEndpoint           string
	skipOcneInstall         bool
	tigeraImageTag          string
	useNodePvEncryption     bool
	verrazzanoResource      string
	verrazzanoTag           string
	verrazzanoVersion       string
	nodeShape               string
	numWorkerNodes          int
	applyYAMLs              string

	// Initialized during before suite, and used across helper functions
	rancherURL        string
	httpClient        *retryablehttp.Client
	cloudCredentialID string
)

// Grabs info from environment variables required by this test suite
func ensureOCNEDriverVarsInitialized() {
	// mandatory environment variables
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

	// optional overrides
	dockerRootDir = getEnvFallback("DOCKER_ROOT_DIR", "/var/lib/docker")
	enableClusterAlerting = getEnvFallbackBool("ENABLE_CLUSTER_ALERTING", false)
	enableClusterMonitoring = getEnvFallbackBool("ENABLE_CLUSTER_MONITORING", false)
	enableNetworkPolicy = getEnvFallbackBool("ENABLE_NETWORK_POLICY", false)
	windowsPreferedCluster = getEnvFallbackBool("WINDOWS_PREFERRED_CLUSTER", false)
	calicoImagePath = getEnvFallback("CALICO_IMAGE_PATH", "olcne")
	clusterCidr = getEnvFallback("CLUSTER_CIDR", "10.96.0.0/16")
	controlPlaneMemoryGbs = getEnvFallbackInt("CONTROL_PLANE_MEMORY_GBS", 16)
	controlPlaneOcpus = getEnvFallbackInt("CONTROL_PLANE_OCPUS", 2)
	controlPlaneShape = getEnvFallback("CONTROL_PLANE_SHAPE", "VM.Standard.E4.Flex")
	controlPlaneVolumeGbs = getEnvFallbackInt("CONTROL_PLANE_VOLUME_GBS", 100)
	corednsImageTag = getEnvFallback("CORE_DNS_IMAGE_TAG", "v1.9.3")
	etcdImageTag = getEnvFallback("ETCD_IMAGE_TAG", "3.5.6")
	imageDisplayName = getEnvFallback("IMAGE_DISPLAY_NAME", "Oracle-Linux-8.7-2023.05.24-0")
	imageID = getEnvFallback("IMAGE_ID", "")
	installCalico = getEnvFallbackBool("INSTALL_CALICO", true)
	installCcm = getEnvFallbackBool("INSTALL_CCM", true)
	installVerrazzano = getEnvFallbackBool("INSTALL_VERRAZZANO", false)
	kubernetesVersion = getEnvFallback("KUBERNETES_VERSION", "v1.25.7")
	numControlPlaneNodes = getEnvFallbackInt("NUM_CONTROL_PLANE_NODES", 1)
	ocneVersion = getEnvFallback("OCNE_VERSION", "1.6")
	podCidr = getEnvFallback("POD_CIDR", "10.244.0.0/16")
	privateRegistry = getEnvFallback("PRIVATE_REGISTRY", "")
	proxyEndpoint = getEnvFallback("PROXY_ENDPOINT", "")
	skipOcneInstall = getEnvFallbackBool("SKIP_OCNE_INSTALL", false)
	tigeraImageTag = getEnvFallback("TIGERA_IMAGE_TAG", "v1.29.0")
	useNodePvEncryption = getEnvFallbackBool("USE_NODE_PV_ENCRYPTION", true)
	verrazzanoResource = getEnvFallback("VERRAZZANO_RESOURCE",
		"apiVersion: install.verrazzano.io/v1beta1\nkind: Verrazzano\nmetadata:\n  name: managed\n  namespace: default\nspec:\n  profile: managed-cluster")
	verrazzanoTag = getEnvFallback("VERRAZZANO_TAG", "v1.6.0-20230609132620-44e8f4d1")
	verrazzanoVersion = getEnvFallback("VERRAZZANO_VERSION", "1.6.0-4574+44e8f4d1")
	nodeShape = getEnvFallback("NODE_SHAPE", "VM.Standard.E4.Flex")
	numWorkerNodes = getEnvFallbackInt("NUM_WORKER_NODES", 1)
	applyYAMLs = getEnvFallback("APPLY_YAMLS", "")
}

// Returns the value of the desired environment variable,
// but returns a fallback value if the environment variable is not set
func getEnvFallback(envVar, fallback string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvFallbackBool(envVar string, fallback bool) bool {
	value := os.Getenv(envVar)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return boolValue
}

func getEnvFallbackInt(envVar string, fallback int) int {
	value := os.Getenv(envVar)
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return intValue
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
	CalicoImagePath       string   `json:"calicoImagePath"`
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
	roc.OciocneEngineConfig.CalicoImagePath = calicoImagePath
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
