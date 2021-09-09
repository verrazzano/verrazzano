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

// Capture Tag for artifact, 1 from BOM, 1 from image in cluster
type imageError struct {
	bomImageTag     string
	clusterImageTag string
}

func main() {
	var vBom verrazzanoBom                           // BOM from platform operator in struct form
	var imagesInstalled = make(map[string]string)    // Map that contains the images installed into the cluster
	var imageTagErrors = make(map[string]imageError) // Map of image names that match but tags don't  Failure Condition
	var imagesNotFound = make(map[string]imageError) // Map of image names not found in cluster. Informational.  This may be valid based on profile

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
	//	populateMapWithContainerImages()

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

// Get the BOM from the plarform operator in the cluster and build the BOM structure from it
func getBOM(vBom *verrazzanoBom) {
	out, err := exec.Command("kubectl", "get", "pod", "-o", "name", "--no-headers=true", "-n", "verrazzano-install").Output()
	if err != nil {
		log.Fatal(err)
	}

	platformOperatorPodName := string(out)
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
func populateMapWithContainerImages(clusterImageMap map[string]string) {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.containers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	containerImages := string(out)
	containerArray := strings.Split(containerImages, " ")
	populateClusterImageMap(containerArray, clusterImageMap)
}

//  Populate a HashMap with all the initContainer images found in the cluster
//  Currently disabled due to condition that is not handled where there are 2 valid versions of oraclelinux
/*func populateMapWithInitContainerImages(clusterImageMap map[string]string) {
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.initContainers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	initContainerImages := string(out)
	initContainerArray := strings.Split(initContainerImages, " ")
	populateClusterImageMap(initContainerArray, clusterImageMap)
}*/

// Validate the images in the BOM against the images found in the cluster
// It can be
//    Valid, Tags match
//    Not Found, OK based on profile
//    InValid, image tags between BOM and cluster image do not mact
func validateBOM(vBom *verrazzanoBom, clusterImageMap map[string]string, imagesNotFound map[string]imageError, imageTagErrors map[string]imageError) bool {
	for _, component := range vBom.Components {
		//		fmt.Printf("Verrazzano bom Component = %s\n", component)
		for _, subcomponent := range component.Subcomponents {
			//			fmt.Printf("Verrazzano bom SubComponent = %s\n", subcomponent)
			for _, image := range subcomponent.Images {
				//				fmt.Println("************************************************")
				//				fmt.Printf("Verrazzano Image Name = %s\n", image.Image)
				//				fmt.Printf("Verrazzano Image Tag = %s\n", image.Tag)
				if tag, ok := clusterImageMap[image.Image]; ok {
					if tag == image.Tag {
						//	fmt.Println("Image Found and Tag matches")
					} else {
						imageTagErrors[image.Image] = imageError{image.Tag, tag}
						return false
					}
				} else {
					imagesNotFound[image.Image] = imageError{image.Tag, ""}
				}

			}
		}
	}
	return true
}

// Report out the findings
// ImagesNotFound is informational
// imageTagErrors is a failure condition
func reportResults(imagesNotFound map[string]imageError, imageTagErrors map[string]imageError, isBOMValid bool) {
	// Dump Images Not Found to Console, Informational

	fmt.Println("Images From BOM not found in the cluster")
	fmt.Println("----------------------------------------")
	for name, tags := range imagesNotFound {
		fmt.Printf("Image From BOM not found in cluster! Image Name = %s, Tag from BOM = %s\n", name, tags.bomImageTag)
	}
	// Dump Images that don't match BOM, Failure
	if !isBOMValid {
		fmt.Println("BOM Images that don't match Cluster Images")
		fmt.Println("----------------------------------------")
		for name, tags := range imageTagErrors {
			fmt.Printf("Verrazzano Image Check Failure! Image Name = %s, Tag from BOM = %s  Tag from Cluster = %s\n", name, tags.bomImageTag, tags.clusterImageTag)
		}
	}
}

// Build out the cluster image map based off of the container array, filter dups on imageName
func populateClusterImageMap(containerArray []string, clusterImageMap map[string]string) {
	// Loop through containers and populate hashmap

	for _, container := range containerArray {
		begin := strings.LastIndex(container, "/")
		end := len(container)
		containerName := container[begin+1 : end]
		nameTag := strings.Split(containerName, ":")
		if _, ok := clusterImageMap[nameTag[0]]; ok {
			//			fmt.Println("Key  ", nameTag[0], " Already exists")
		} else {
			clusterImageMap[nameTag[0]] = nameTag[1]
		}
	}
}
