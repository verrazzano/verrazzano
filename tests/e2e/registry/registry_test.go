// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 2 * time.Minute
	pollingInterval = 10 * time.Second

	extOSPod                   = "opensearch-cluster-master-0"
	helmOperationPodNamePrefix = "helm-operation"
	shellImage                 = "/shell:"
)

var registry = os.Getenv("REGISTRY")

// Map that contains the images present in the BOM with the associated registry URL
var imageRegistryMap = make(map[string]string)

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
	constants.VerrazzanoSystemNamespace,
	constants.VerrazzanoMonitoringNamespace,
}

var t = framework.NewTestFramework("registry")
var _ = t.BeforeSuite(func() {})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Image Registry Verification", Label("f:platform-lcm.private-registry"),
	func() {
		t.It("All the pods in the cluster have the expected registry URLs",
			func() {
				foundHelmOperationPod := false
				shellImageHasCorrectPrefix := false
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
						// Skip private registry validation in case of external OpenSearch
						if pod.Name == extOSPod {
							continue
						}
						if strings.HasPrefix(pod.Name, helmOperationPodNamePrefix) {
							foundHelmOperationPod = true
						}
						podLabels := pod.GetLabels()
						_, ok := podLabels["job-name"]
						if pod.Status.Phase != corev1.PodRunning && ok {
							continue
						}
						pkg.Log(pkg.Info, fmt.Sprintf("%d. Validating the registry url prefix for pod: %s in namespace: %s", i, pod.Name, ns))
						for k := range pod.Spec.Containers {
							image := pod.Spec.Containers[k].Image
							registryURL, err := getRegistryURL(image)
							Expect(err).To(BeNil(), fmt.Sprintf("Failed to get the expected registry url for image %s: %v", image, err))
							hasCorrectRegistryPrefix := strings.HasPrefix(image, registryURL)

							// When checking the Rancher Helm Operation pod Shell image, there may be old pods left over so we only check to see
							// if at least one of the Shell images has the correct registry prefix
							if strings.Contains(image, shellImage) {
								if hasCorrectRegistryPrefix {
									shellImageHasCorrectPrefix = true
								}
								continue
							}
							Expect(hasCorrectRegistryPrefix).To(BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in containers, doesn't start with expected registry URL prefix %s, image name %s", pod.Name, registryURL, image))
						}
						for k := range pod.Spec.InitContainers {
							image := pod.Spec.InitContainers[k].Image
							registryURL, err := getRegistryURL(image)
							Expect(err).To(BeNil(), fmt.Sprintf("Failed to get the expected registry url for init container image %s: %v", image, err))
							Expect(strings.HasPrefix(image, registryURL)).To(BeTrue(),
								fmt.Sprintf("FAIL: The image for the pod %s in initContainers, doesn't start with expected registry URL prefix %s, image name %s", pod.Name, registryURL, image))
						}
					}
				}

				// If we found at least one Rancher Helm Operation pod, then make sure at least one of the Shell images has the correct prefix
				if foundHelmOperationPod {
					Expect(shellImageHasCorrectPrefix).To(BeTrue(), "FAIL: Found at least one Rancher Helm Operation pod but none of the shell images has the expected registry prefix")
				}
			})
	})

// getRegistryURL returns the private registry url if the private registry env is set
// If private registry is not set, the registry url is determined based on what is specified in the BOM
// for a given image
func getRegistryURL(containerImage string) (string, error) {
	// For private registry, determine the registry url from the corresponding env variables
	if len(registry) > 0 {
		return pkg.GetImagePrefix(), nil
	}
	// Populate image registry map if not already done
	if len(imageRegistryMap) == 0 {
		err := populateImageRegistryMap()
		if err != nil {
			return "", err
		}
	}
	imageName := getImageName(containerImage)
	// If the image is not defined in the bom, return an error
	if imageRegistryMap[imageName] == "" {
		return "", fmt.Errorf("the image %s is not specified in the BOM from platform operator", imageName)
	}
	registryURLFromBom := imageRegistryMap[imageName]
	// If the registry of the image is docker.io and the container image does not have the registry prefix,
	// remove docker.io from the constructed registry url. This is mainly to address the case where some of the
	// images such as rancher-webhook does not have "docker.io" prefix in the image in the pod output.
	if strings.HasPrefix(registryURLFromBom, "docker.io") && !strings.HasPrefix(containerImage, "docker.io") {
		return strings.TrimPrefix(registryURLFromBom, "docker.io/"), nil
	}
	return registryURLFromBom, nil
}

// Populate image registry map from BOM
func populateImageRegistryMap() error {
	// Get the BOM from installed Platform Operator
	bomDoc, err := pkg.GetBOMDoc()
	if err != nil {
		return err
	}
	globalRegistry := bomDoc.Registry
	for _, component := range bomDoc.Components {
		for _, subComponent := range component.SubComponents {
			registry := globalRegistry
			if len(subComponent.Registry) > 0 {
				registry = subComponent.Registry
			}
			repository := subComponent.Repository
			for _, image := range subComponent.Images {
				if len(image.Registry) > 0 {
					registry = image.Registry
				}
				if len(image.Repository) > 0 {
					repository = image.Repository
				}
				imageRegistryMap[image.ImageName] = registry + "/" + repository
			}
		}
	}
	return nil
}

// Get the name of the image from the image URL
func getImageName(imageURL string) string {
	imageWithoutVer := strings.Split(imageURL, ":")[0]
	imageParts := strings.Split(imageWithoutVer, "/")
	return imageParts[len(imageParts)-1]
}
