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

var vBom verrazzanoBom

var imagesInstalled = make(map[string]string)

type imageError struct {
	bomImageTag     string
	clusterImageTag string
}

var imageTagErrors = make(map[string]imageError)

var imagesNotFound = make(map[string]imageError)

var failure bool = false

func main() {
	// Validate KubeConfig
	validateKubeConfig()

	// Get the BOM from installed Platform Operator
	getBOM()

	// Get the container images in the cluster
	out, err := exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.containers[*].image}\"").Output()
	if err != nil {
		log.Fatal(err)
	}

	containerImages := string(out)
	containerArray := strings.Split(containerImages, " ")
	populateClusterImageMap(containerArray)

	// Get the initContainer images in the cluster
	/*	out, err = exec.Command("kubectl", "get", "pods", "--all-namespaces", "-o", "jsonpath=\"{.items[*].spec.initContainers[*].image}\"").Output()
		if err != nil {
			log.Fatal(err)
		}

		initContainerImages := string(out)
		initContainerArray := strings.Split(initContainerImages, " ")
		populateClusterImageMap(initContainerArray) */

	//  Loop through BOM and check against cluster images
	for _, component := range vBom.Components {
		//		fmt.Printf("Verrazzano bom Component = %s\n", component)
		for _, subcomponent := range component.Subcomponents {
			//			fmt.Printf("Verrazzano bom SubComponent = %s\n", subcomponent)
			for _, image := range subcomponent.Images {
				//				fmt.Println("************************************************")
				//				fmt.Printf("Verrazzano Image Name = %s\n", image.Image)
				//				fmt.Printf("Verrazzano Image Tag = %s\n", image.Tag)
				if tag, ok := imagesInstalled[image.Image]; ok {
					if tag == image.Tag {
						//	fmt.Println("Image Found and Tag matches")
					} else {
						imageTagErrors[image.Image] = imageError{image.Tag, tag}
						failure = true
					}
				} else {
					imagesNotFound[image.Image] = imageError{image.Tag, ""}
				}

			}
		}
	}

	// Dump Images Not Found to Console, Informational
	fmt.Println("Images From BOM not found in the cluster")
	fmt.Println("----------------------------------------")
	for name, tags := range imagesNotFound {
		fmt.Printf("Image From BOM not found in cluster! Image Name = %s, Tag from BOM = %s\n", name, tags.bomImageTag)
	}
	// Dump Images that don't match BOM, Failure
	if failure {
		fmt.Println("BOM Images that don't match Cluster Images")
		fmt.Println("----------------------------------------")
		for name, tags := range imageTagErrors {
			fmt.Printf("Verrazzano Image Check Failure! Image Name = %s, Tag from BOM = %s  Tag from Cluster = %s\n", name, tags.bomImageTag, tags.clusterImageTag)
		}
		os.Exit(1)
	}

	//Success
	os.Exit(0)
}

func validateKubeConfig() {
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
		os.Setenv("KUBECONFIG", kubeconfig)
	} else {
		fmt.Println("KUBECONFIG Not Valued, Terminating")
		os.Exit(1)
	}
}

func getBOM() {
	out, err := exec.Command("kubectl", "get", "pod", "-o", "name", "--no-headers=true", "-n", "verrazzano-install").Output()
	if err != nil {
		log.Fatal(err)
	}

	var platform_operator_pod_name string
	platform_operator_pod_name = string(out)
	platform_operator_pod_name = strings.TrimSuffix(platform_operator_pod_name, "\n")
	fmt.Printf("The platform operator pod name is %s\n", platform_operator_pod_name)

	//  Get the BOM from platform-operator
	out, err = exec.Command("kubectl", "exec", "-it", string(platform_operator_pod_name), "-n", "verrazzano-install", "--", "cat", "/verrazzano/platform-operator/verrazzano-bom.json").Output()
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal([]byte(out), &vBom)

}

func populateClusterImageMap(containerArray []string) {
	// Loop through containers and populate hashmap

	for _, container := range containerArray {
		begin := strings.LastIndex(container, "/")
		end := len(container)
		containerName := container[begin+1 : end]
		//		fmt.Println(i, containerName)
		nameTag := strings.Split(containerName, ":")
		if _, ok := imagesInstalled[nameTag[0]]; ok {
			//			fmt.Println("Key  ", nameTag[0], " Already exists")
		} else {
			imagesInstalled[nameTag[0]] = nameTag[1]
		}
	}
}
