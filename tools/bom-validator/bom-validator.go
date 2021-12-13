// Copyright (c) 2021, Oracle and/or its affiliates.
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
)

const tagLen = 10                                                          // The number of unique tags for a specific image
const platformOperatorPodNameSearchString = "verrazzano-platform-operator" // Pod Substring for finding the platform operator pod

// Struct based on Verrazzano BOM JSON
type verrazzanoBom struct {
	Registry   string `json:"registry"`
	Version    string `json:"version"`
	Components []struct {
		Name          string `json:"name"`
		Subcomponents []struct {
			Repository string `json:"repository"`
			Name       string `json:"name"`
			Images     []struct {
				Image            string `json:"image"`
				Tag              string `json:"tag"`
				HelmFullImageKey string `json:"helmFullImageKey"`
			} `json:"images"`
		} `json:"subcomponents"`
	} `json:"components"`
}

// Capture Tags for artifact, 1 from BOM, All from images in cluster
type imageError struct {
	bomImageTag      string
	clusterImageTags [tagLen]string
}

var ignoreSubComponents = []string{
	"additional-rancher",
}

func main() {
	var vBom verrazzanoBom                                // BOM from platform operator in struct form
	var imagesInstalled = make(map[string][tagLen]string) // Map that contains the images installed into the cluster with associated set of tags
	var imageTagErrors = make(map[string]imageError)      // Map of image names that match but tags don't  Failure Condition
	var imagesNotFound = make(map[string]string)          // Map of image names not found in cluster. Informational.  This may be valid based on profile

	// Validate KubeConfig
	if !validateKubeConfig() {
		fmt.Println("KUBECONFIG Not Valued, Terminating")
		os.Exit(1)
	}

	// Get the BOM from installed Platform Operator
	getBOM(&vBom)

	// Get the container images in the cluster
	populateMapWithContainerImages(imagesInstalled)

	// Get the initContainer images in the cluster
	populateMapWithInitContainerImages(imagesInstalled)

	//  Loop through BOM and check against cluster images
	isBOMValid := validateBOM(&vBom, imagesInstalled, imagesNotFound, imageTagErrors)

	// Write to stdout
	reportResults(imagesNotFound, imageTagErrors, isBOMValid)

	//Failure
	if !isBOMValid {
		os.Exit(1)
	}
}

// Validate that KubeConfig is valued. This will point to the cluster being validated
func validateKubeConfig() bool {
	var kubeconfig string
	// Get the options
	kubeconfigPtr := flag.String("kubeconfig", "", "KubeConfig for cluster being validated")
	flag.Parse()
	if *kubeconfigPtr == "" {
		// Try the Env
		kubeconfig = os.Getenv("KUBECONFIG")
	} else {
		kubeconfig = *kubeconfigPtr
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

	json.Unmarshal([]byte(out), &vBom)

}

// Populate a HashMap with all the container images found in the cluster
func populateMapWithContainerImages(clusterImageMap map[string][tagLen]string) {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.containers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	containerImages := string(out)
	containerArray := strings.Split(containerImages, " ")
	populateClusterImageMap(containerArray, clusterImageMap)
}

//  Populate a HashMap with all the initContainer images found in the cluster
func populateMapWithInitContainerImages(clusterImageMap map[string][tagLen]string) {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.initContainers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	initContainerImages := string(out)
	initContainerArray := strings.Split(initContainerImages, " ")
	populateClusterImageMap(initContainerArray, clusterImageMap)
}

// Validate the images in the BOM against the images found in the cluster
// It can be
//    Valid, Tags match
//    Not Found, OK based on profile
//    InValid, image tags between BOM and cluster image do not match
func validateBOM(vBom *verrazzanoBom, clusterImageMap map[string][tagLen]string, imagesNotFound map[string]string, imageTagErrors map[string]imageError) bool {
	var errorsFound bool = false
	for _, component := range vBom.Components {
		for _, subcomponent := range component.Subcomponents {
			if ignoreSubComponent(subcomponent.Name) {
				fmt.Printf("Subcomponent %s of component %s on ignore list, skipping images %v\n", subcomponent.Name, component.Name, subcomponent.Images)
				continue
			}
			for _, image := range subcomponent.Images {
				if tags, ok := clusterImageMap[image.Image]; ok {
					var tagFound bool = false
					for _, tag := range tags {
						if tag == image.Tag {
							tagFound = true
							break
						}
					}
					if !tagFound {
						imageTagErrors[image.Image] = imageError{image.Tag, tags} // TODO  Fix up message
						errorsFound = true
					}
				} else {
					imagesNotFound[image.Image] = image.Tag
				}

			}
		}
	}
	return !errorsFound
}

//ignoreSubComponent - checks to see if a particular subcomponent is to be ignored
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
func reportResults(imagesNotFound map[string]string, imageTagErrors map[string]imageError, isBOMValid bool) {
	// Dump Images Not Found to Console, Informational

	fmt.Println("Images From BOM not found in the cluster")
	fmt.Println("----------------------------------------")
	for name, tag := range imagesNotFound {
		fmt.Printf("Image From BOM not found in cluster! Image Name = %s, Tag from BOM = %s\n", name, tag)
	}
	// Dump Images that don't match BOM, Failure
	if !isBOMValid {
		fmt.Println("BOM Images that don't match Cluster Images")
		fmt.Println("----------------------------------------")
		for name, tags := range imageTagErrors {
			fmt.Printf("Verrazzano Image Check Failure! Image Name = %s, Tag from BOM = %s  Tag from Cluster = %+v\n", name, tags.bomImageTag, tags.clusterImageTags)
		}
	}
}

// Build out the cluster image map based off of the container array, filter dups
func populateClusterImageMap(containerArray []string, clusterImageMap map[string][tagLen]string) {
	// Loop through containers and populate hashmap
	var i int
	for _, container := range containerArray {
		begin := strings.LastIndex(container, "/")
		end := len(container)
		containerName := container[begin+1 : end]
		nameTag := strings.Split(containerName, ":")
		tags := clusterImageMap[nameTag[0]]
		for i = 0; i < len(tags); i++ {
			if tags[i] == nameTag[1] {
				break
			}
			if tags[i] == "" {
				tags[i] = nameTag[1]
				clusterImageMap[nameTag[0]] = tags
				break
			}
		}
		// Notify that the tag array for a specific image has maxed out.  This should not happen.  There should not be a Verrazzano installed image that has more than 10(taglen) distinct tags
		if i == tagLen {
			fmt.Printf("Tag Limit Exceeded! More than 10 tags exist for Image Name = %s, Tags from Cluster = %+v\n", nameTag[0], tags)
		}
	}
}
