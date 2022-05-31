// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	vzstring "github.com/verrazzano/verrazzano/pkg/string"
)

const (
	platformOperatorPodNameSearchString = "verrazzano-platform-operator" // Pod Substring for finding the platform operator pod
	oracleLinuxWarningMessage           = "See case-03/04 of VZ-5962, generalisations of bom validator"
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
	bomImageTags []string
}

var (
	ignoreSubComponents []string
	kubeconfig          string
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
	"coredns":     	   {alternateTags: []string{"v1.8.0"}, message: oracleLinuxWarningMessage},
	"kindnetd":        {alternateTags: []string{"v20210326-1e038dc5"}, message: oracleLinuxWarningMessage},
	"kube-apiserver":  {alternateTags: []string{"v1.21.1"}, message: oracleLinuxWarningMessage},
	"kube-controller-manager":     {alternateTags: []string{"v1.21.1"}, message: oracleLinuxWarningMessage},
	"kube-scheduler":     {alternateTags: []string{"v1.21.1"}, message: oracleLinuxWarningMessage},
	"controller":     {alternateTags: []string{"v0.11.0"}, message: oracleLinuxWarningMessage},
	"speaker":     {alternateTags: []string{"v0.11.0"}, message: oracleLinuxWarningMessage},
	"etcd":     {alternateTags: []string{"3.4.13-0"}, message: oracleLinuxWarningMessage},
	"oraclelinux":     {alternateTags: []string{"7-slim", "7.9"}, message: oracleLinuxWarningMessage},
}

func main() {
	var vBom verrazzanoBom                                // BOM from platform operator in struct form
	var imagesInstalled = make(map[string][]string) // Map that contains the images installed into the cluster with associated set of tags
	var containersInstalled =  make(map[string]bool)
	var imageTagErrors = make(map[string]imageError)      // Map of image names that match but tags don't  Failure Condition
	var imagesNotFound = make(map[string]string)          // Map of image names not found in cluster. Informational.  This may be valid based on profile
	var imageWarnings = make(map[string]string)           // Map of image names not found in cluster. Informational.  This may be valid based on profile

	// Validate KubeConfig
	if !validateKubeConfig() {
		fmt.Println("KUBECONFIG Not Valued, Terminating")
		os.Exit(1)
	}

	// Get the BOM from installed Platform Operator
	getBOM(&vBom)

	// populate the bom's containers and images into map
	populateBomContainerImagesMap(&vBom,containersInstalled, imagesInstalled)

	// Validate the cluster's container images with the populated bom images and tags
	validateClusterContainerImages(imagesInstalled, containersInstalled, imagesNotFound, imageTagErrors, imageWarnings)

	// Validate the cluster's init container images with the populated bom images and tags
	validateClusterInitContainerImages(imagesInstalled, containersInstalled, imagesNotFound, imageTagErrors, imageWarnings)

	// Checkout the results
	errorFound := reportResults(imagesNotFound, imageTagErrors, imageWarnings)

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

func populateBomContainerImagesMap(vBom *verrazzanoBom, clusterContainerMap map[string]bool, clusterImageMap map[string][]string) {
	for _, component := range vBom.Components {
		for _, subcomponent := range component.Subcomponents {
			for _, image := range subcomponent.Images {
				clusterContainerMap[image.Image+":"+image.Tag] = true
				clusterImageMap[image.Image] = append(clusterImageMap[image.Image], image.Tag)
			}
		}
	}
}

// Populate a HashMap with all the container images found in the cluster
func validateClusterContainerImages(clusterImageMap map[string][]string, clusterContainerMap map[string]bool, imagesNotFound map[string]string,
	imageTagErrors map[string]imageError, imageWarnings map[string]string) {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.containers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	containerImages := string(out)
	containerArray := strings.Split(containerImages, " ")
	validateBomImages(containerArray, clusterImageMap, clusterContainerMap,imagesNotFound,imageTagErrors,imageWarnings)
}

//  Populate a HashMap with all the initContainer images found in the cluster
func validateClusterInitContainerImages(clusterImageMap map[string][]string, clusterContainerMap map[string]bool, imagesNotFound map[string]string,
	imageTagErrors map[string]imageError, imageWarnings map[string]string) {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.initContainers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	initContainerImages := string(out)
	initContainerArray := strings.Split(initContainerImages, " ")
	validateBomImages(initContainerArray, clusterImageMap, clusterContainerMap,imagesNotFound,imageTagErrors,imageWarnings)
}

// ignoreSubComponent - checks to see if a particular subcomponent is to be ignored
func ignoreSubComponent(name string) bool {
	for _, subComp := range ignoreSubComponents {
		if subComp == name {
			return true
		}
	}
	return false
}

// Report out the findings
// ImagesNotFound is informational
// imageTagErrors is a failure condition
func reportResults(imagesNotFound map[string]string, imageTagErrors map[string]imageError, warnings map[string]string) bool{
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
	if len(imagesNotFound) > 0 {
		fmt.Println()
		fmt.Println("Image Errors: CLuster Images not mentioned into bom")
		fmt.Println(textDivider)
		for name, tag := range imagesNotFound {
			fmt.Printf("Image not mentioned into bom: %s:%s\n", name, tag)
		}
		error = true
	}
	if len(imageTagErrors) > 0 {
		fmt.Println()
		fmt.Println("Image's Tag Errors: Cluster Image's Tags doesn't mentioned into Bom Image's Tags")
		fmt.Println(textDivider)
		for name, tags := range imageTagErrors {
			fmt.Println("Check failed! Image Name = ", name, ", Tag from Cluster = ", tags.clusterImageTag, "Tags from Bom = ", tags.bomImageTags)
		}
		error = true
	}
	return error
}

// Build out the cluster image map based off of the container array, filter dups
func validateBomImages(containerArray []string, clusterImageMap map[string][]string, clusterContainerMap map[string]bool, imagesNotFound map[string]string,
	imageTagErrors map[string]imageError, imageWarnings map[string]string) {
	for _, container := range containerArray {
		begin := strings.LastIndex(container, "/")
		end := len(container)
		containerName := container[begin+1 : end]
		nameTag := strings.Split(containerName, ":")

		// Check if the image/tag in the cluster is known to have issues
		imageWarning, hasKnownIssues := knownImageIssues[nameTag[0]]
		if hasKnownIssues && vzstring.SliceContainsString(imageWarning.alternateTags, nameTag[1]) {
			imageWarnings[nameTag[0]] = fmt.Sprintf("Known issue for image %s, found tag %s, expected tag %s message: %s",
				nameTag[0], nameTag[1], clusterImageMap[nameTag[0]], imageWarning.message)
			continue
		}
		// error scenarios,
		// 1. if cluster's image not found into bom
		if _, ok := clusterImageMap[nameTag[0]]; !ok {
			// cluster's image not found into bom
			imagesNotFound[nameTag[0]] = nameTag[1]
			continue
		}
		// 2. if cluster's image's tag not found into bom
		if !clusterContainerMap[containerName] {
			// cluster's image found into bom but,
			// cluster's image:tag not found into bom
			imageTagErrors[nameTag[0]] = imageError{nameTag[1], clusterImageMap[nameTag[0]]}
		}
	}
}
