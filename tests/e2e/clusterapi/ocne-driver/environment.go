// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"go.uber.org/zap"
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
func ensureOCNEDriverVarsInitialized(log *zap.SugaredLogger) error {
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
	clusterCidr = getEnvFallback("CLUSTER_CIDR", "10.96.0.0/16")
	controlPlaneMemoryGbs = getEnvFallbackInt("CONTROL_PLANE_MEMORY_GBS", 16)
	controlPlaneOcpus = getEnvFallbackInt("CONTROL_PLANE_OCPUS", 2)
	controlPlaneShape = getEnvFallback("CONTROL_PLANE_SHAPE", "VM.Standard.E4.Flex")
	controlPlaneVolumeGbs = getEnvFallbackInt("CONTROL_PLANE_VOLUME_GBS", 100)
	imageID = getEnvFallback("IMAGE_ID", "")
	installCalico = getEnvFallbackBool("INSTALL_CALICO", true)
	installCcm = getEnvFallbackBool("INSTALL_CCM", true)
	installVerrazzano = getEnvFallbackBool("INSTALL_VERRAZZANO", false)
	numControlPlaneNodes = getEnvFallbackInt("NUM_CONTROL_PLANE_NODES", 1)
	podCidr = getEnvFallback("POD_CIDR", "10.244.0.0/16")
	privateRegistry = getEnvFallback("PRIVATE_REGISTRY", "")
	proxyEndpoint = getEnvFallback("PROXY_ENDPOINT", "")
	skipOcneInstall = getEnvFallbackBool("SKIP_OCNE_INSTALL", false)
	useNodePvEncryption = getEnvFallbackBool("USE_NODE_PV_ENCRYPTION", true)
	verrazzanoResource = getEnvFallback("VERRAZZANO_RESOURCE",
		"apiVersion: install.verrazzano.io/v1beta1\nkind: Verrazzano\nmetadata:\n  name: managed\n  namespace: default\nspec:\n  profile: managed-cluster")
	nodeShape = getEnvFallback("NODE_SHAPE", "VM.Standard.E4.Flex")
	numWorkerNodes = getEnvFallbackInt("NUM_WORKER_NODES", 1)
	applyYAMLs = getEnvFallback("APPLY_YAMLS", "")
	if err := fillOCNEMetadata(); err != nil {
		return err
	}
	if err := fillOCNEVersion(); err != nil {
		return err
	}
	if err := fillNodeImage(); err != nil {
		return err
	}
	if err := fillVerrazzanoVersions(log); err != nil {
		return err
	}
	return nil
}

// Initializes variables from OCNE metadata ConfigMap. Values are optionally overridden.
func fillOCNEMetadata() error {
	// TODO: Use ocne-metadata configmap to get fallback values
	corednsImageTag = getEnvFallback("CORE_DNS_IMAGE_TAG", "v1.9.3")
	etcdImageTag = getEnvFallback("ETCD_IMAGE_TAG", "3.5.6")
	tigeraImageTag = getEnvFallback("TIGERA_IMAGE_TAG", "v1.29.0")
	kubernetesVersion = getEnvFallback("KUBERNETES_VERSION", "v1.25.7")
	return nil
}

// Initializes OCNE Version, optionally overridden.
func fillOCNEVersion() error {
	// TODO: Use Rancher API call to get fallback value
	ocneVersion = getEnvFallback("OCNE_VERSION", "1.6")
	return nil
}

// Initializes the node image, optionally overridden
func fillNodeImage() error {
	// TODO: Use Rancher API call to get fallback value
	imageDisplayName = getEnvFallback("IMAGE_DISPLAY_NAME", "Oracle-Linux-8.7-2023.05.24-0")
	return nil
}

// Initializes the VZ version and tag for the created OCNE clusters. Values are optionally overridden.
func fillVerrazzanoVersions(log *zap.SugaredLogger) error {
	var vzTagFallback, vzVersionFallback string

	// Use Rancher API call to get fallback values
	requestURL, adminToken := setupRequest(rancherURL, "/meta/ocne/verrazzanoVersions", log)
	response, err := helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error requesting Verrazzano versions from Rancher: %s", err)
		return err
	}
	responseMap := response.ChildrenMap()
	if len(responseMap) != 1 {
		err = fmt.Errorf("response to GET call to /meta/ocne/verrazzanoVersions does not have expected length")
		log.Error(err)
		return err
	}
	for version, tag := range responseMap {
		vzTagFallback = tag.Data().(string)
		vzVersionFallback = version
	}

	// Initialize values
	verrazzanoTag = getEnvFallback("VERRAZZANO_TAG", vzTagFallback)
	verrazzanoVersion = getEnvFallback("VERRAZZANO_VERSION", vzVersionFallback)
	return nil
}

// TODO: possibly get node shapes from Rancher API

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
