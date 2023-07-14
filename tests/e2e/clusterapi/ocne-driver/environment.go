// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

// Initializes the variables required to create a cloud credential
func ensureVarsInitializedForCredential() {
	region = os.Getenv("OCI_REGION")
	userID = os.Getenv("OCI_USER_ID")
	tenancyID = os.Getenv("OCI_TENANCY_ID")
	fingerprint = os.Getenv("OCI_CREDENTIALS_FINGERPRINT")
	privateKeyPath = os.Getenv("OCI_PRIVATE_KEY_PATH")
	ocneClusterNameSuffix = os.Getenv("OCNE_CLUSTER_NAME_SUFFIX")
}

// Grabs info from environment variables required by this test suite.
// Requires an existing cloud credential.
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
	numWorkerNodes = getEnvFallbackInt("NUM_WORKER_NODES", 1)
	applyYAMLs = getEnvFallback("APPLY_YAMLS", "")
	if err := fillOCNEMetadata(log); err != nil {
		return err
	}
	if err := fillOCNEVersion(log); err != nil {
		return err
	}
	if err := fillNodeImage(log); err != nil {
		return err
	}
	if err := fillVerrazzanoVersions(log); err != nil {
		return err
	}
	if err := fillNodeShapes(log); err != nil {
		return err
	}
	return nil
}

// Initializes variables from OCNE metadata ConfigMap. Values are optionally overridden.
func fillOCNEMetadata(log *zap.SugaredLogger) error {
	var coreDNSFallback, etcdFallback, tigeraFallback, kubernetesFallback string

	// Use ocne-metadata configmap to get fallback values
	const ocneMetadataCMName = "ocne-metadata"
	cm, err := pkg.GetConfigMap(ocneMetadataCMName, "verrazzano-capi")
	if err != nil {
		log.Errorf("error getting %s ConfigMap: %s", ocneMetadataCMName, err)
		return err
	}
	if cm == nil {
		err = fmt.Errorf("%s ConfigMap not found", ocneMetadataCMName)
		log.Error(err)
		return err
	}
	dataYaml := cm.Data["mapping"]
	var mapToContents map[string]interface{}
	if err = yaml.Unmarshal([]byte(dataYaml), &mapToContents); err != nil {
		log.Errorf("yaml unmarshalling error: %s", err)
		return err
	}
	if len(mapToContents) != 1 {
		err = fmt.Errorf("data inside %s ConfigMap not formatted as expcted", ocneMetadataCMName)
		log.Error(err)
		return err
	}
	for k8sVersion, contents := range mapToContents {
		kubernetesFallback = k8sVersion
		coreDNSFallback = contents.(map[string]interface{})["container-images"].(map[string]interface{})["coredns"].(string)
		etcdFallback = contents.(map[string]interface{})["container-images"].(map[string]interface{})["etcd"].(string)
		tigeraFallback = contents.(map[string]interface{})["container-images"].(map[string]interface{})["tigera-operator"].(string)
	}

	// Initialize values
	corednsImageTag = getEnvFallback("CORE_DNS_IMAGE_TAG", coreDNSFallback)
	etcdImageTag = getEnvFallback("ETCD_IMAGE_TAG", etcdFallback)
	tigeraImageTag = getEnvFallback("TIGERA_IMAGE_TAG", tigeraFallback)
	kubernetesVersion = getEnvFallback("KUBERNETES_VERSION", kubernetesFallback)
	return nil
}

// Initializes OCNE Version, optionally overridden.
func fillOCNEVersion(log *zap.SugaredLogger) error {
	var ocneVersionFallback string

	// Use Rancher API call to get fallback value
	requestURL, adminToken := setupRequest(rancherURL, "/meta/ocne/ocneVersions", log)
	response, err := helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error requesting OCNE version from Rancher: %s", err)
		return err
	}
	versionList := response.Children()
	if len(versionList) != 1 {
		err = fmt.Errorf("response to OCNE versions request does not have expected length")
		log.Error(err)
		return err
	}
	ocneVersionFallback = versionList[0].Data().(string)

	// Initialize value
	ocneVersion = getEnvFallback("OCNE_VERSION", ocneVersionFallback)
	return nil
}

// Initializes the node image, optionally overridden
func fillNodeImage(log *zap.SugaredLogger) error {
	var linuxImageFallback string

	// Use Rancher API call to get fallback value
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("/meta/oci/nodeImages?cloudCredentialId=%s", cloudCredentialID), log)
	response, err := helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error requesting node images from Rancher: %s", err)
		return err
	}
	imageList := response.Children()
	for _, image := range imageList {
		imageString := image.Data().(string)
		// filter a suitable OL 8 image, same as what the rancher UI does
		if strings.HasPrefix(imageString, "Oracle-Linux-8") && !strings.Contains(imageString, "aarch64") {
			linuxImageFallback = imageString
			break
		}
	}
	if linuxImageFallback == "" {
		err = fmt.Errorf("could not find a suitable node image")
		log.Error(err)
		return err
	}

	// Initialize value
	imageDisplayName = getEnvFallback("IMAGE_DISPLAY_NAME", linuxImageFallback)
	return nil
}

// Initializes the VZ version and tag for the created OCNE clusters, optionally overridden.
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
		err = fmt.Errorf("response to Verrazzano versions request does not have expected length")
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

// Initializes the control plane and worker node shapes, optionally overridden.
func fillNodeShapes(log *zap.SugaredLogger) error {
	var cpShapeFallback, nodeShapeFallback string

	// Use Rancher API call to get fallback values
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("/meta/oci/nodeShapes?cloudCredentialId=%s", cloudCredentialID), log)
	response, err := helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error requesting node shapes from Rancher: %s", err)
		return err
	}
	shapeList := response.Children()
	if len(shapeList) == 0 {
		err = fmt.Errorf("request for node shapes to Rancher API returned an empty list")
		log.Error(err)
		return err
	}
	// If the list contains "VZ.Standard.E4.Flex", default to that, similar to the Rancher UI.
	// Otherwise, use the first image in the list.
	cpShapeFallback = shapeList[0].Data().(string)
	nodeShapeFallback = shapeList[0].Data().(string)
	for _, shape := range shapeList {
		shapeString := shape.Data().(string)
		if shapeString == "VM.Standard.E4.Flex" {
			cpShapeFallback = shapeString
			nodeShapeFallback = shapeString
			break
		}
	}

	// Initialize values
	controlPlaneShape = getEnvFallback("CONTROL_PLANE_SHAPE", cpShapeFallback)
	nodeShape = getEnvFallback("NODE_SHAPE", nodeShapeFallback)
	return nil
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
