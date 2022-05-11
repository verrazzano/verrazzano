// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 2 * time.Minute
	pollingInterval = 10 * time.Second
	// Pod Substring for finding the platform operator pod
	platformOperatorPodNameSearchString = "verrazzano-platform-operator"
)

var registry = os.Getenv("REGISTRY")
var imagePrefix = pkg.GetImagePrefix()

// Map that contains the images present in the BOM with the associated registry URL
var imageRegistryMap = make(map[string]string)

// Struct based on Verrazzano BOM JSON
type verrazzanoBom struct {
	Registry   string `json:"registry"`
	Version    string `json:"version"`
	Components []struct {
		Name          string `json:"name"`
		Subcomponents []struct {
			Registry   string `json:"registry"`
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

// List of namespaces from which all the pods are queried to confirm the images are loaded from the target registry/repo
var listOfNamespaces = []string{
	"cattle-global-data",
	"cattle-global-data-nt",
	"cattle-system",
	"cert-manager",
	"default",
	"fleet-default",
	"fleet-local",
	"fleet-system",
	"ingress-nginx",
	"istio-system",
	"keycloak",
	"local",
	"monitoring",
	"verrazzano-install",
	"verrazzano-mc",
	"verrazzano-system",
}

var t = framework.NewTestFramework("registry")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Image Registry Verification", Label("f:platform-lcm.private-registry"),
	func() {
		t.It("All the pods in the cluster have the expected registry URLs",
			func() {
				var pod corev1.Pod
				for i, ns := range listOfNamespaces {
					var pods *corev1.PodList
					Eventually(func() (*corev1.PodList, error) {
						var err error
						pods, err = pkg.ListPods(ns, metav1.ListOptions{})
						return pods, err
					}, waitTimeout, pollingInterval).ShouldNot(BeNil(), fmt.Sprintf("Error listing pods in the namespace %s", ns))

					for j := range pods.Items {
						pod = pods.Items[j]
						pkg.Log(pkg.Info, fmt.Sprintf("%d. Validating the registry url prefix for pod: %s in namespace: %s", i, pod.Name, ns))
						for k := range pod.Spec.Containers {
							registryURL, err := getRegistryURL(pod.Spec.Containers[k].Image)
							Expect(err).To(BeNil(), fmt.Sprintf("Failed to get the expected registry url for image %s: %v",
								pod.Spec.Containers[k].Image, err))
							Expect(strings.HasPrefix(pod.Spec.Containers[k].Image, registryURL)).To(BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in containers, doesn't start with expected registry URL prefix %s, image name %s", pod.Name, registryURL, pod.Spec.Containers[k].Image))
						}
						for k := range pod.Spec.InitContainers {
							registryURL, err := getRegistryURL(pod.Spec.InitContainers[k].Image)
							Expect(err).To(BeNil(), fmt.Sprintf("Failed to get the expected registry url for init container image %s: %v",
								pod.Spec.InitContainers[k].Image, err))
							Expect(strings.HasPrefix(pod.Spec.InitContainers[k].Image, registryURL)).To(BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in initContainers, doesn't start with expected registry URL prefix %s, image name %s", pod.Name, registryURL, pod.Spec.InitContainers[k].Image))
						}
					}
				}
			})
	})

// getRegistryURL returns the private registry url if the private registry env is set
// If private registry not set, the registry url is determined based on what is specified in the BOM
func getRegistryURL(image string) (string, error) {
	// If it is private registry
	if len(registry) > 0 {
		return imagePrefix, nil
	}
	// Get the images and the corresponding registries from the BOM
	if len(imageRegistryMap) == 0 {
		err := populateImageRegistryMap()
		if err != nil {
			return "", err
		}
	}
	// Remove the version from image
	imageURL := strings.Split(image, ":")[0]
	imageParts := strings.Split(imageURL, "/")
	imageName := imageParts[len(imageParts)-1]
	// If the image is not defined in the bom, return the default registry
	if imageRegistryMap[imageName] == "" {
		return imagePrefix, nil
	}
	imageURLFromBom := imageRegistryMap[imageName] + "/" + imageName
	imagePartsFromBom := strings.Split(imageURLFromBom, "/")
	if len(imageParts) != len(imagePartsFromBom) {
		// If docker.io is not specified in the container image, remove docker.io from the constructed registry url
		if (imagePartsFromBom[0] == "docker.io") && pkg.SlicesContainSameStrings(imagePartsFromBom[1:], imageParts) {
			imagePartsFromBom = imagePartsFromBom[1:]
		}
	}
	// Remove image name
	return strings.Join(imagePartsFromBom[:len(imagePartsFromBom)-1], "/"), nil
}

// Populate image registries map from BOM
func populateImageRegistryMap() error {
	var vBom verrazzanoBom
	// Get the BOM from installed Platform Operator
	err := getBOM(&vBom)
	if err != nil {
		return err
	}
	globalRegistry := vBom.Registry
	for _, component := range vBom.Components {
		for _, subComponent := range component.Subcomponents {
			for _, image := range subComponent.Images {
				registry := globalRegistry
				if len(subComponent.Registry) > 0 {
					registry = subComponent.Registry
				}
				imageRegistryMap[image.Image] = registry + "/" + subComponent.Repository
			}
		}
	}
	return nil
}

// Get the BOM from the platform operator in the cluster and build the BOM structure from it
func getBOM(vBom *verrazzanoBom) error {
	var platformOperatorPodName = ""

	out, err := exec.Command("kubectl", "get", "pod", "-o", "name", "--no-headers=true", "-n", "verrazzano-install").Output()
	if err != nil {
		return fmt.Errorf("error in gettting %s pod name: %v", platformOperatorPodNameSearchString, err)
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
		return fmt.Errorf("platform operator pod name not found in verrazzano-install namespace")
	}

	platformOperatorPodName = strings.TrimSuffix(platformOperatorPodName, "\n")
	fmt.Printf("Getting the registry details in BOM from the platform operator %s\n", platformOperatorPodName)

	//  Get the BOM from platform-operator
	out, err = exec.Command("kubectl", "exec", "-it", platformOperatorPodName, "-n", "verrazzano-install", "--",
		"cat", "/verrazzano/platform-operator/verrazzano-bom.json").Output()
	if err != nil {
		return err
	}
	if len(string(out)) == 0 {
		return fmt.Errorf("error retrieving BOM from platform operator, zero length")
	}

	json.Unmarshal(out, &vBom)
	return nil
}
