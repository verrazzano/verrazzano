// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bomvalidator

import (
	"encoding/json"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
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

// BOM validations validates the images of below allowed namespaces only
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

var vBom verrazzanoBom                                  // BOM from platform operator in struct form
var clusterImageArray []string                          // List of cluster installed images
var bomImages = make(map[string][]string)               // Map of images mentioned into the BOM with associated set of tags
var clusterImageTagErrors = make(map[string]imageError) // Map of cluster image tags doesn't match with BOM, hence a Failure Condition
var clusterImagesNotFound = make(map[string]string)     // Map of cluster image doesn't match with BOM, hence a Failure Condition
var clusterImageWarnings = make(map[string]string)      // Map of image names not found in cluster. Warning/ Known Issues/ Informational.  This may be valid based on profile

var t = framework.NewTestFramework("BOM validator")

var _ = t.AfterSuite(func() {})
var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("BOM Validator", Label("f:platform-lcm.install"), func() {
	t.Context("Post VZ Installations", func() {
		t.It("Has Successful BOM Validation Report", func() {
			Eventually(validateKubeConfig).Should(BeTrue())
			getBOM()
			Expect(vBom.Components).NotTo(BeNil())
			populateBomContainerImagesMap()
			Expect(bomImages).NotTo(BeEmpty())
			populateClusterContainerImages()
			Expect(clusterImageArray).NotTo(BeEmpty())
			Eventually(scanClusterImagesWithBom).Should(BeTrue())
			Eventually(BomValidationReport).Should(BeTrue())
		})
	})
})

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
func getBOM() {
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

// Populate BOM images into Hashmap bomImages
// bomImages contains a map of "image" in the BOM to validate an image found in an allowed namespace exists in the BOM
func populateBomContainerImagesMap() {
	for _, component := range vBom.Components {
		for _, subcomponent := range component.Subcomponents {
			for _, image := range subcomponent.Images {
				bomImages[image.Image] = append(bomImages[image.Image], image.Tag)
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
// Send Cluster's Images Array 'A' for BOM Validations against populated BOM hashmap 'bomImages'
// Hashmap 'clusterImagesNotFound' are images found in allowed namespaces that are not declared in the BOM
// Hashmap 'clusterImageTagErrors' are images in allowed namespaces without matching tags in the BOM
func populateClusterContainerImages() {
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
	clusterImageArray = strings.Split(strings.TrimSpace(containerImages), " ")
}

// Report out the findings
// clusterImagesNotFound is a failure condition
// clusterImageTagErrors is a failure condition
func BomValidationReport() bool {
	// Dump Images Not Found to Console, Informational
	const textDivider = "----------------------------------------"

	if len(clusterImageWarnings) > 0 {
		fmt.Println()
		fmt.Println("Image Warnings - Tags not at expected BOM level due to known issues")
		fmt.Println(textDivider)
		for name, msg := range clusterImageWarnings {
			fmt.Printf("Warning: Image Name = %s: %s\n", name, msg)
		}
	}
	if len(clusterImagesNotFound) > 0 {
		fmt.Println()
		fmt.Println("Image Errors: Images found in allowed namespaces not declared in BOM")
		fmt.Println(textDivider)
		for name, tag := range clusterImagesNotFound {
			fmt.Printf("Found image in allowed namespace not declared in BOM : %s:%s\n", name, tag)
		}
		return false
	}
	if len(clusterImageTagErrors) > 0 {
		fmt.Println()
		fmt.Println("Image Errors: Images found in allowed namespace of cluster with unexpected tags")
		fmt.Println(textDivider)
		for name, tags := range clusterImageTagErrors {
			fmt.Println("Check failed! Image Name = ", name, ", Tag from Cluster = ", tags.clusterImageTag, "Tags from BOM = ", tags.bomImageTags)
		}
		return false
	}
	fmt.Println()
	fmt.Println("!! BOM Validation Successful !!")
	return true
}

// Validate out the presence of cluster images and tags into vz BOM
func scanClusterImagesWithBom() bool {
	for _, container := range clusterImageArray {
		begin := strings.LastIndex(container, "/")
		end := len(container)
		containerName := container[begin+1 : end]
		nameTag := strings.Split(containerName, ":")

		// Check if the image/tag in the cluster is known to have issues
		imageWarning, hasKnownIssues := knownImageIssues[nameTag[0]]
		if hasKnownIssues && vzstring.SliceContainsString(imageWarning.alternateTags, nameTag[1]) {
			clusterImageWarnings[nameTag[0]] = fmt.Sprintf("Known issue for image %s, found tag %s, expected tag %s message: %s",
				nameTag[0], nameTag[1], bomImages[nameTag[0]], imageWarning.message)
			continue
		}
		// error scenarios,
		// 1. if cluster's image not found into BOM's image map
		if _, ok := bomImages[nameTag[0]]; !ok {
			// cluster's image not found into BOM
			clusterImagesNotFound[nameTag[0]] = nameTag[1]
			continue
		}
		// 2. if cluster's image's version (tag) mismatched to BOM's image versions(tags)
		if !vzstring.SliceContainsString(bomImages[nameTag[0]], nameTag[1]) {
			// cluster's image's version (tag) mismatched to BOM image versions(tags)
			clusterImageTagErrors[nameTag[0]] = imageError{nameTag[1], bomImages[nameTag[0]]}
		}
	}
	// validation went successful
	return true
}
