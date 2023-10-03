// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package okecapidriver

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
)

var (
	// Initialized by ensureOKEDriverVarsInitialized, required environment variables
	region                   string
	vcnID                    string
	userID                   string
	tenancyID                string
	fingerprint              string
	privateKeyPath           string
	nodePublicKeyPath        string
	compartmentID            string
	workerNodeSubnet         string
	controlPlaneSubnet       string
	loadBalancerSubnet       string
	okeCapiClusterNameSuffix string

	// Initialized by ensureOKEDriverVarsInitialized, optional overrides
	dockerRootDir           string
	enableClusterAlerting   bool
	enableClusterMonitoring bool
	enableNetworkPolicy     bool
	windowsPreferedCluster  bool
	clusterCidr             string
	imageDisplayName        string
	imageID                 string
	kubernetesVersion       string
	podCidr                 string
	nodeShape               string
	applyYAMLs              string
	installVerrazzano       bool

	okeSupportedVersions []string

	// Initialized during before suite, and used across helper functions
	rancherURL        string
	httpClient        *retryablehttp.Client
	cloudCredentialID string

	okeMetadataItemToInstall OKEMetadataItem
	okeMetadataItemToUpgrade OKEMetadataItem
)

// Verify required Environment Variables are set
func verifyRequiredEnvironmentVariables() {
	region = pkg.GetRequiredEnvVarOrFail("OCI_REGION")
	userID = pkg.GetRequiredEnvVarOrFail("OCI_USER_ID")
	tenancyID = pkg.GetRequiredEnvVarOrFail("OCI_TENANCY_ID")
	fingerprint = pkg.GetRequiredEnvVarOrFail("OCI_CREDENTIALS_FINGERPRINT")
	privateKeyPath = pkg.GetRequiredEnvVarOrFail("OCI_PRIVATE_KEY_PATH")
	okeCapiClusterNameSuffix = pkg.GetRequiredEnvVarOrFail("OKE_CAPI_CLUSTER_NAME_SUFFIX")
	vcnID = pkg.GetRequiredEnvVarOrFail("OCI_VCN_ID")
	nodePublicKeyPath = pkg.GetRequiredEnvVarOrFail("NODE_PUBLIC_KEY_PATH")
	compartmentID = pkg.GetRequiredEnvVarOrFail("OCI_COMPARTMENT_ID")
	workerNodeSubnet = pkg.GetRequiredEnvVarOrFail("WORKER_NODE_SUBNET")
	controlPlaneSubnet = pkg.GetRequiredEnvVarOrFail("CONTROL_PLANE_SUBNET")
	loadBalancerSubnet = pkg.GetRequiredEnvVarOrFail("LOAD_BALANCER_SUBNET")
}

// Grabs info from optional environment variables.
// Requires an existing cloud credential.
func ensureOKEDriverVarsInitialized(log *zap.SugaredLogger) error {
	// optional overrides
	dockerRootDir = pkg.GetEnvFallback("DOCKER_ROOT_DIR", "/var/lib/docker")
	enableClusterAlerting = pkg.GetEnvFallbackBool("ENABLE_CLUSTER_ALERTING", false)
	enableClusterMonitoring = pkg.GetEnvFallbackBool("ENABLE_CLUSTER_MONITORING", false)
	enableNetworkPolicy = pkg.GetEnvFallbackBool("ENABLE_NETWORK_POLICY", false)
	windowsPreferedCluster = pkg.GetEnvFallbackBool("WINDOWS_PREFERRED_CLUSTER", false)
	clusterCidr = pkg.GetEnvFallback("CLUSTER_CIDR", "10.96.0.0/16")
	imageID = pkg.GetEnvFallback("IMAGE_ID", "")
	podCidr = pkg.GetEnvFallback("POD_CIDR", "10.244.0.0/16")
	applyYAMLs = pkg.GetEnvFallback("APPLY_YAMLS", "")
	installVerrazzano = pkg.GetEnvFallbackBool("INSTALL_VERRAZZANO_ON_CAPI", false)
	if err := fillOKEMetadata(log); err != nil {
		return err
	}
	if err := fillNodeImage(log); err != nil {
		return err
	}
	if err := fillNodeShapes(log); err != nil {
		return err
	}
	return nil
}

// Initializes variables from OKE metadata ConfigMap. Values are optionally overridden.
func fillOKEMetadata(log *zap.SugaredLogger) error {
	// Initialize values
	kubernetesVersion = pkg.GetEnvFallback("KUBERNETES_VERSION", "v1.26.2")
	okeSupportedVersions = strings.Split(pkg.GetEnvFallback("OKE_VERSIONS", "v1.27.2, v1.26.7, v1.26.2, v1.25.12, v1.25.4"), ",")

	for _, k8sVersion := range okeSupportedVersions {
		k8sSemVerFallback, err := semver.NewVersion(strings.TrimSpace(k8sVersion))
		if err != nil {
			log.Errorf("kubernetes version parsing error: %s", err)
			return err
		}
		// finding the minimum kubernetes version to install a OKE cluster
		if okeMetadataItemToInstall.KubernetesVersion == nil || k8sSemVerFallback.LessThan(okeMetadataItemToInstall.KubernetesVersion) {
			okeMetadataItemToInstall = OKEMetadataItem{KubernetesVersion: k8sSemVerFallback}
		}
		// finding the maximum kubernetes version to update the OKE cluster
		if okeMetadataItemToUpgrade.KubernetesVersion == nil || k8sSemVerFallback.GreaterThan(okeMetadataItemToUpgrade.KubernetesVersion) {
			okeMetadataItemToUpgrade = OKEMetadataItem{KubernetesVersion: k8sSemVerFallback}
		}
	}
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

// Initializes the control plane and worker node shapes, optionally overridden.
func fillNodeShapes(log *zap.SugaredLogger) error {
	var nodeShapeFallback string

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
	nodeShapeFallback = shapeList[0].Data().(string)
	for _, shape := range shapeList {
		shapeString := shape.Data().(string)
		if shapeString == "VM.Standard.E4.Flex" {
			nodeShapeFallback = shapeString
			break
		}
	}

	// Initialize values
	nodeShape = pkg.GetEnvFallback("NODE_SHAPE", nodeShapeFallback)
	return nil
}
