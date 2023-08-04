// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"net/http"
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

	ocneMetadataItemToInstall OCNEMetadataItem
	ocneMetadataItemToUpgrade OCNEMetadataItem
)

// Verify required Environment Variables are set
func verifyRequiredEnvironmentVariables() {
	region = pkg.GetRequiredEnvVarOrFail("OCI_REGION")
	userID = pkg.GetRequiredEnvVarOrFail("OCI_USER_ID")
	tenancyID = pkg.GetRequiredEnvVarOrFail("OCI_TENANCY_ID")
	fingerprint = pkg.GetRequiredEnvVarOrFail("OCI_CREDENTIALS_FINGERPRINT")
	privateKeyPath = pkg.GetRequiredEnvVarOrFail("OCI_PRIVATE_KEY_PATH")
	ocneClusterNameSuffix = pkg.GetRequiredEnvVarOrFail("OCNE_CLUSTER_NAME_SUFFIX")
	vcnID = pkg.GetRequiredEnvVarOrFail("OCI_VCN_ID")
	nodePublicKeyPath = pkg.GetRequiredEnvVarOrFail("NODE_PUBLIC_KEY_PATH")
	compartmentID = pkg.GetRequiredEnvVarOrFail("OCI_COMPARTMENT_ID")
	workerNodeSubnet = pkg.GetRequiredEnvVarOrFail("WORKER_NODE_SUBNET")
	controlPlaneSubnet = pkg.GetRequiredEnvVarOrFail("CONTROL_PLANE_SUBNET")
	loadBalancerSubnet = pkg.GetRequiredEnvVarOrFail("LOAD_BALANCER_SUBNET")
}

// Grabs info from optional environment variables.
// Requires an existing cloud credential.
func ensureOCNEDriverVarsInitialized(log *zap.SugaredLogger) error {
	// optional overrides
	dockerRootDir = pkg.GetEnvFallback("DOCKER_ROOT_DIR", "/var/lib/docker")
	enableClusterAlerting = pkg.GetEnvFallbackBool("ENABLE_CLUSTER_ALERTING", false)
	enableClusterMonitoring = pkg.GetEnvFallbackBool("ENABLE_CLUSTER_MONITORING", false)
	enableNetworkPolicy = pkg.GetEnvFallbackBool("ENABLE_NETWORK_POLICY", false)
	windowsPreferedCluster = pkg.GetEnvFallbackBool("WINDOWS_PREFERRED_CLUSTER", false)
	clusterCidr = pkg.GetEnvFallback("CLUSTER_CIDR", "10.96.0.0/16")
	controlPlaneMemoryGbs = pkg.GetEnvFallbackInt("CONTROL_PLANE_MEMORY_GBS", 16)
	controlPlaneOcpus = pkg.GetEnvFallbackInt("CONTROL_PLANE_OCPUS", 2)
	controlPlaneVolumeGbs = pkg.GetEnvFallbackInt("CONTROL_PLANE_VOLUME_GBS", 100)
	imageID = pkg.GetEnvFallback("IMAGE_ID", "")
	installCalico = pkg.GetEnvFallbackBool("INSTALL_CALICO", true)
	installCcm = pkg.GetEnvFallbackBool("INSTALL_CCM", true)
	installVerrazzano = pkg.GetEnvFallbackBool("INSTALL_VERRAZZANO", false)
	numControlPlaneNodes = pkg.GetEnvFallbackInt("NUM_CONTROL_PLANE_NODES", 1)
	podCidr = pkg.GetEnvFallback("POD_CIDR", "10.244.0.0/16")
	privateRegistry = pkg.GetEnvFallback("PRIVATE_REGISTRY", "")
	proxyEndpoint = pkg.GetEnvFallback("PROXY_ENDPOINT", "")
	skipOcneInstall = pkg.GetEnvFallbackBool("SKIP_OCNE_INSTALL", false)
	useNodePvEncryption = pkg.GetEnvFallbackBool("USE_NODE_PV_ENCRYPTION", true)
	verrazzanoResource = pkg.GetEnvFallback("VERRAZZANO_RESOURCE", `apiVersion: install.verrazzano.io/v1beta1
kind: Verrazzano
metadata:
  name: managed
  namespace: default
spec:
  profile: managed-cluster
  components:
	ingressNGINX:
	  overrides:
		- values:
			controller:
			  service:
				annotations:
				  service.beta.kubernetes.io/oci-load-balancer-shape : "flexible"
				  service.beta.kubernetes.io/oci-load-balancer-shape-flex-min: "10"
				  service.beta.kubernetes.io/oci-load-balancer-shape-flex-max: "100"
	  type: LoadBalancer
	istio:
	  overrides:
		- values:
			apiVersion: install.istio.io/v1alpha1
			kind: IstioOperator
			spec:
			  values:
				gateways:
				  istio-ingressgateway:
					serviceAnnotations:
					  service.beta.kubernetes.io/oci-load-balancer-shape: "flexible"
					  service.beta.kubernetes.io/oci-load-balancer-shape-flex-min: "10"
					  service.beta.kubernetes.io/oci-load-balancer-shape-flex-max: "100"`)
	numWorkerNodes = pkg.GetEnvFallbackInt("NUM_WORKER_NODES", 1)
	applyYAMLs = pkg.GetEnvFallback("APPLY_YAMLS", "")
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
	// Unmarshal dataYaml into a map,
	// since the first and only top-level field's key is an unknown Kubernetes version
	dataYaml := cm.Data["mapping"]
	var mapToContents map[string]interface{}
	if err = yaml.Unmarshal([]byte(dataYaml), &mapToContents); err != nil {
		log.Errorf("yaml unmarshalling error: %s", err)
		return err
	}
	if len(mapToContents) < 1 {
		err = fmt.Errorf("data inside %s ConfigMap not formatted as expcted", ocneMetadataCMName)
		log.Error(err)
		return err
	}

	for k8sVersion, contents := range mapToContents {
		// Retrieve the Kubernetes version
		kubernetesFallback = k8sVersion

		// Convert the inner contents to a Golang struct for easier access
		var contentStruct OCNEMetadataContents
		contentBytes, err := yaml.Marshal(contents)
		if err != nil {
			log.Errorf("yaml marshalling error: %s", err)
			return err
		}
		if err = yaml.Unmarshal(contentBytes, &contentStruct); err != nil {
			log.Errorf("yaml unmarshalling error: %s", err)
			return err
		}
		coreDNSFallback = contentStruct.ContainerImages.Coredns
		etcdFallback = contentStruct.ContainerImages.Etcd
		tigeraFallback = contentStruct.ContainerImages.TigeraOperator

		k8sSemVerFallback, err := semver.NewVersion(k8sVersion)
		if err != nil {
			log.Errorf("kubernetes version parsing error: %s", err)
			return err
		}
		// finding the minimum kubernetes version to install a OCNE cluster
		if ocneMetadataItemToInstall.KubernetesVersion == nil || k8sSemVerFallback.LessThan(ocneMetadataItemToInstall.KubernetesVersion) {
			ocneMetadataItemToInstall = OCNEMetadataItem{KubernetesVersion: k8sSemVerFallback, OCNEMetadataContents: contentStruct}
		}
		// finding the maximum kubernetes version to update the OCNE cluster
		if ocneMetadataItemToUpgrade.KubernetesVersion == nil || k8sSemVerFallback.GreaterThan(ocneMetadataItemToUpgrade.KubernetesVersion) {
			ocneMetadataItemToUpgrade = OCNEMetadataItem{KubernetesVersion: k8sSemVerFallback, OCNEMetadataContents: contentStruct}
		}
	}

	// Initialize values
	corednsImageTag = pkg.GetEnvFallback("CORE_DNS_IMAGE_TAG", coreDNSFallback)
	etcdImageTag = pkg.GetEnvFallback("ETCD_IMAGE_TAG", etcdFallback)
	tigeraImageTag = pkg.GetEnvFallback("TIGERA_IMAGE_TAG", tigeraFallback)
	kubernetesVersion = pkg.GetEnvFallback("KUBERNETES_VERSION", kubernetesFallback)
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
	if len(versionList) < 1 {
		err = fmt.Errorf("response to OCNE versions request does not have expected length")
		log.Error(err)
		return err
	}
	ocneVersionFallback = versionList[0].Data().(string)

	// Initialize value
	ocneVersion = pkg.GetEnvFallback("OCNE_VERSION", ocneVersionFallback)
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
	imageDisplayName = pkg.GetEnvFallback("IMAGE_DISPLAY_NAME", linuxImageFallback)
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
	if len(responseMap) < 1 {
		err = fmt.Errorf("response to Verrazzano versions request does not have expected length")
		log.Error(err)
		return err
	}
	for version, tag := range responseMap {
		vzTagFallback = tag.Data().(string)
		vzVersionFallback = version
	}

	// Initialize values
	verrazzanoTag = pkg.GetEnvFallback("VERRAZZANO_TAG", vzTagFallback)
	verrazzanoVersion = pkg.GetEnvFallback("VERRAZZANO_VERSION", vzVersionFallback)
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
	controlPlaneShape = pkg.GetEnvFallback("CONTROL_PLANE_SHAPE", cpShapeFallback)
	nodeShape = pkg.GetEnvFallback("NODE_SHAPE", nodeShapeFallback)
	return nil
}
