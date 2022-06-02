// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	platformOperatorPodNameSearchString = "verrazzano-platform-operator"                        // Pod Substring for finding the platform operator pod
	rancherWarningMessage               = "See VZ-5937, Rancher upgrade issue, all VZ versions" // For known Rancher issues with VZ upgrade
)

// Verrazzano BOM types

type imageDetails struct {
	Image            string `json:"image"`
	Tag              string `json:"tag"`
	HelmFullImageKey string `json:"helmFullImageKey"`
}

type subComponentType struct {
	Repository string         `json:"repository"`
	Name       string         `json:"name"`
	Images     []imageDetails `json:"images"`
}

type componentType struct {
	Name          string             `json:"name"`
	Subcomponents []subComponentType `json:"subcomponents"`
}

type verrazzanoBom struct {
	Registry   string          `json:"registry"`
	Version    string          `json:"version"`
	Components []componentType `json:"components"`
}

// Capture Tags for artifact, 1 from BOM, All from images in cluster
type imageError struct {
	clusterImageTag string
	bomImageTags    []string
}

var (
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "KubeConfig for cluster being validated")
	flag.Parse()
}

// Hack to work around an issue with the 1.2 upgrade; Rancher does not always update the webhook image
type knownIssues struct {
	alternateTags []string
	message       string
}

// Mainly a workaround for Rancher additional images; Rancher does not always update to the latest version
// in the BOM file, possible Rancher bug that we are pursuing with the Rancher team
var knownImageIssues = map[string]knownIssues{
	"rancher-webhook": {alternateTags: []string{"v0.1.1", "v0.1.2", "v0.1.4"}, message: rancherWarningMessage},
	"fleet-agent":     {alternateTags: []string{"v0.3.5"}, message: rancherWarningMessage},
	"fleet":           {alternateTags: []string{"v0.3.5"}, message: rancherWarningMessage},
	"gitjob":          {alternateTags: []string{"v0.1.15"}, message: rancherWarningMessage},
	"shell":           {alternateTags: []string{"v0.1.6"}, message: rancherWarningMessage},
}

// Bom validations validates the images of below allowed namespaces only
var allowedNamespaces = []string{
	"^cattle-*",
	"^fleet-*",
	"^cluster-fleet-*",
	"^cert-manager",
	"^ingress-nginx",
	"^istio-system",
	"^keycloak",
	"^monitoring",
	"^verrazzano-*",
}

func main() {
	var vBom verrazzanoBom                                  // BOM from platform operator in struct form
	var bomImages = make(map[string][]string)               // Map that contains the images mentioned into the bom with associated set of tags
	var clusterImageTagErrors = make(map[string]imageError) // Map of cluster image match but tags doesn't match with bom, hence a Failure Condition
	var clusterImagesNotFound = make(map[string]string)     // Map of cluster image doesn't match with bom, hence a Failure Condition
	var clusterImageWarnings = make(map[string]string)      // Map of image names not found in cluster. Warning/ Known Issues/ Informational.  This may be valid based on profile

	// Validate KubeConfig
	if !validateKubeConfig() {
		fmt.Println("KUBECONFIG Not Valued, Terminating")
		os.Exit(1)
	}
	// Get the BOM from installed Platform Operator
	getBOM(&vBom)
	// populate the bom's images and their versions into hashmap
	populateBomContainerImagesMap(&vBom, bomImages)
	// Validate the cluster's container (including init containers) images with the populated bom image map
	validateClusterContainerImages(bomImages, clusterImagesNotFound, clusterImageTagErrors, clusterImageWarnings)
	// Report the bom validation results
	errorFound := reportResults(clusterImagesNotFound, clusterImageTagErrors, clusterImageWarnings)
	// Failure
	if errorFound {
		os.Exit(1)
	}
}

// Validate that KubeConfig is valued. This will point to the cluster being validated
func validateKubeConfig() bool {
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if kubeconfig != "" {
		fmt.Println("USING KUBECONFIG: ", kubeconfig)
		return true
	}
	return false
}

// Get the BOM from the platform operator in the cluster and build the BOM structure from it
func getBOM(vBom *verrazzanoBom) {
	var platformOperatorPodName string = ""

	out, err := exec.Command("kubectl", "get", "pod", "-o", "name", "--no-headers=true", "-n", "verrazzano-install").Output()
	if err != nil {
		log.Fatal(err)
	}

	vzInstallPods := string(out)
	vzInstallPodArray := strings.Split(vzInstallPods, "\n")
	for _, podName := range vzInstallPodArray {
		if strings.Contains(podName, platformOperatorPodNameSearchString) {
			platformOperatorPodName = podName
			break
		}
	}
	if platformOperatorPodName == "" {
		log.Fatal("Platform Operator Pod Name not found in verrazzano-install namespace!")
	}

	platformOperatorPodName = strings.TrimSuffix(platformOperatorPodName, "\n")
	fmt.Printf("The platform operator pod name is %s\n", platformOperatorPodName)

	//  Get the BOM from platform-operator
	out, err = exec.Command("kubectl", "exec", "-it", platformOperatorPodName, "-n", "verrazzano-install", "--", "cat", "/verrazzano/platform-operator/verrazzano-bom.json").Output()
	if err != nil {
		log.Fatal(err)
	}
	if len(string(out)) == 0 {
		log.Fatal("Error retrieving BOM from platform operator, zero length\n")
	}
	json.Unmarshal(out, &vBom)
}

// Populate bom images into Hashmap bomImageMap
// bomImageMap contains a map of "image" in the BOM to validate an image found in an allowed namespace exists in the bom
func populateBomContainerImagesMap(vBom *verrazzanoBom, bomImageMap map[string][]string) {
	for _, component := range vBom.Components {
		for _, subcomponent := range component.Subcomponents {
			for _, image := range subcomponent.Images {
				bomImageMap[image.Image] = append(bomImageMap[image.Image], image.Tag)
			}
		}
	}
}

// Return all installed cluster namespaces
func getAllNamespaces() []string {
	cmd := "kubectl get namespaces | grep -v NAME | awk '{print $1}'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// Get the cluster namespaces and validate images of allowed namespaces only
// Populate an Array 'A' with all the container & initContainer images found in the cluster of allowed namespaces
// Send Cluster's Images Array 'A' for BOM Validations against populated bom hashmap 'bomImageMap'
// Hashmap 'clusterImagesNotFound' are images found in allowed namespaces that are not declared in the BOM
// Hashmap 'clusterImageTagErrors' are images in allowed namespaces without matching tags in the BOM
func validateClusterContainerImages(bomImageMap map[string][]string, clusterImagesNotFound map[string]string,
	clusterImageTagErrors map[string]imageError, clusterImageWarnings map[string]string) {
	var containerImages string
	installedClusterNamespaces := getAllNamespaces()
	for _, installedNamespace := range installedClusterNamespaces {
		for _, whiteListedNamespace := range allowedNamespaces {
			if ok, _ := regexp.MatchString(whiteListedNamespace, installedNamespace); ok {
				out, err := exec.Command("kubectl", "get", "pods", "-n", installedNamespace, "-o", "jsonpath=\"{.items[*].spec.containers[*].image}\"").Output()
				if err != nil {
					log.Fatal(err)
				}
				containerImages += strings.TrimPrefix(strings.TrimSuffix(string(out), `"`), `"`)
				out, err = exec.Command("kubectl", "get", "pods", "-n", installedNamespace, "-o", "jsonpath=\"{.items[*].spec.initContainers[*].image}\"").Output()
				if err != nil {
					log.Fatal(err)
				}
				containerImages += strings.TrimPrefix(strings.TrimSuffix(string(out), `"`), `"`)
			}
		}
	}
	// List of all the container & initContainer images found in the cluster
	clusterImageArray := strings.Split(strings.TrimSpace(containerImages), " ")
	// validating cluster images with bom images
	validateContainerImages(clusterImageArray, bomImageMap, clusterImagesNotFound, clusterImageTagErrors, clusterImageWarnings)
}

// Report out the findings
// clusterImagesNotFound is a failure condition
// clusterImageTagErrors is a failure condition
func reportResults(clusterImagesNotFound map[string]string, clusterImageTagErrors map[string]imageError, warnings map[string]string) bool {
	// Dump Images Not Found to Console, Informational
	const textDivider = "----------------------------------------"
	var error bool = false

	if len(warnings) > 0 {
		fmt.Println()
		fmt.Println("Image Warnings - Tags not at expected BOM level due to known issues")
		fmt.Println(textDivider)
		for name, msg := range warnings {
			fmt.Printf("Warning: Image Name = %s: %s\n", name, msg)
		}
	}
	if len(clusterImagesNotFound) > 0 {
		fmt.Println()
		fmt.Println("Image Errors: Images found in allowed namespaces not declared in BOM")
		fmt.Println(textDivider)
		for name, tag := range clusterImagesNotFound {
			fmt.Printf("Found image in allowed namespace not declared in bom: %s:%s\n", name, tag)
		}
		error = true
	}
	if len(clusterImageTagErrors) > 0 {
		fmt.Println()
		fmt.Println("Image Errors: Images found in allowed namespace of cluster with unexpected tags")
		fmt.Println(textDivider)
		for name, tags := range clusterImageTagErrors {
			fmt.Println("Check failed! Image Name = ", name, ", Tag from Cluster = ", tags.clusterImageTag, "Tags from Bom = ", tags.bomImageTags)
		}
		error = true
	}
	if !error {
		fmt.Println()
		fmt.Println("!! BOM Validation Successful !!")
	}
	return error
}

// Validate out the presence of cluster images and tags into vz bom
func validateContainerImages(clusterImageArray []string, bomImageMap map[string][]string, clusterImagesNotFound map[string]string,
	clusterImageTagErrors map[string]imageError, clusterImageWarnings map[string]string) {
	for _, container := range clusterImageArray {
		begin := strings.LastIndex(container, "/")
		end := len(container)
		containerName := container[begin+1 : end]
		nameTag := strings.Split(containerName, ":")

		// Check if the image/tag in the cluster is known to have issues
		imageWarning, hasKnownIssues := knownImageIssues[nameTag[0]]
		if hasKnownIssues && vzstring.SliceContainsString(imageWarning.alternateTags, nameTag[1]) {
			clusterImageWarnings[nameTag[0]] = fmt.Sprintf("Known issue for image %s, found tag %s, expected tag %s message: %s",
				nameTag[0], nameTag[1], bomImageMap[nameTag[0]], imageWarning.message)
			continue
		}
		// error scenarios,
		// 1. if cluster's image not found into bom's image map
		if _, ok := bomImageMap[nameTag[0]]; !ok {
			// cluster's image not found into bom
			clusterImagesNotFound[nameTag[0]] = nameTag[1]
			continue
		}
		// 2. if cluster's image's version (tag) mismatched to bom's image versions(tags)
		if !vzstring.SliceContainsString(bomImageMap[nameTag[0]], nameTag[1]) {
			// cluster's image's version (tag) mismatched to bom image versions(tags)
			clusterImageTagErrors[nameTag[0]] = imageError{nameTag[1], bomImageMap[nameTag[0]]}
		}
	}
}
