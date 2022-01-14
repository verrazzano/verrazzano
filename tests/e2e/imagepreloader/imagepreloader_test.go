// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagepreloader

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v8obom "github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// imageState - information about each docker image being loaded
type imageState struct {
	name   string
	loaded bool
}

const testNamespace string = "image-preloader"
const testName string = "preloader-test"
const bomPath string = "../../../platform-operator/verrazzano-bom.json"

// The examples images are not contained with a BOM.  For now extract the image names from the example files.
// exampleApp - information about the image name of each application and where to find it
type exampleApp struct {
	containerName string
	imageName     string
	exampleFile   string
}

var exampleImages = []exampleApp{
	{"example-roberts-coherence", "container-registry.oracle.com/verrazzano/example-roberts-coherence", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"example-roberts-helidon-stock-application", "container-registry.oracle.com/verrazzano/example-roberts-helidon-stock-application", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"example-bobbys-coherence", "container-registry.oracle.com/verrazzano/example-bobbys-coherence", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"example-bobbys-helidon-stock-application", "container-registry.oracle.com/verrazzano/example-bobbys-helidon-stock-application", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"weblogic", "container-registry.oracle.com/middleware/weblogic", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"example-bobbys-front-end", "container-registry.oracle.com/verrazzano/example-bobbys-front-end", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"example-bobs-books-order-manager", "container-registry.oracle.com/verrazzano/example-bobs-books-order-manager", "../../../examples/bobs-books/bobs-books-comp.yaml"},
	{"example-helidon-greet-app-v1", "ghcr.io/verrazzano/example-helidon-greet-app-v1", "../../../examples/hello-helidon/hello-helidon-comp.yaml"},
	{"example-springboot", "ghcr.io/verrazzano/example-springboot", "../../../examples/springboot-app/springboot-comp.yaml"},
	{"example-todo", "container-registry.oracle.com/verrazzano/example-todo", "../../../examples/todo-list/todo-list-components.yaml"},
}

var bom v8obom.Bom
var imageList map[string]*imageState
var podName string
var kubeconfig string

var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortWaitTimeout = 5 * time.Minute
var shortPollingInterval = 10 * time.Second

var t = framework.NewTestFramework("imagepreloader")
var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {
	pkg.Log(pkg.Info, fmt.Sprintf("Create namespace %s", testNamespace))
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(testNamespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Parse Verrazzano bom file")
	var err error
	bom, err = v8obom.NewBom(bomPath)
	Expect(err).ToNot(HaveOccurred())
	pkg.Log(pkg.Info, fmt.Sprintf("The Bom version is %s", bom.GetVersion()))
})

var _ = t.AfterSuite(func() {
	if failed {
		err := pkg.ExecuteClusterDumpWithEnvVarConfig()
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("Failed to dump cluster with error: %v", err))
		}
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Delete the namespace %s", testNamespace))
	Eventually(func() error {
		return pkg.DeleteNamespace(testNamespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, fmt.Sprintf("Wait for the namespace %s to be terminated", testNamespace))
	Eventually(func() bool {
		_, err := pkg.GetNamespace(testNamespace)
		if err != nil {
			return errors.IsNotFound(err)
		}
		return false
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	// Show the summary of images preloaded
	for _, image := range imageList {
		pkg.Log(pkg.Info, fmt.Sprintf("Image: %s, Loaded: %t", image.name, image.loaded))
	}
})

var _ = t.Describe("Load Verrazzano Container Images", func() {
	start := time.Now()
	t.Context("Deployment", func() {
		t.It("deploy test pod", func() {
			err := deployDaemonSet(testName, testNamespace)
			Expect(err).ToNot(HaveOccurred())
		})
		t.It("wait for the pod to be ready", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(testNamespace, []string{testName})
			}, shortWaitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("%s failed to deploy", testName))
		})
		t.It("get the pod name", func() {
			var podsList []corev1.Pod
			Eventually(func() error {
				var err error
				podsList, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"name": testName}}, testNamespace)
				if err != nil && errors.IsNotFound(err) {
					// Ignore pods not found
					return nil
				}
				return err
			}, shortWaitTimeout, pollingInterval).ShouldNot(HaveOccurred())
			Expect(podsList).ShouldNot(BeNil())
			Expect(len(podsList)).To(Equal(1))
			podName = podsList[0].Name
			Expect(podName).ToNot(BeNil())
		})
	})
	t.Context("Use ephemeral containers to inject images", func() {
		It("create container list", func() {
			var err error
			imageList, err = createImageList(bom)
			Expect(err).ToNot(HaveOccurred())
		})
		t.It(fmt.Sprintf("inject images into the test pod %s", podName), func() {
			// Get the kubeconfig location
			var err error
			kubeconfig, err = k8sutil.GetKubeConfigLocation()
			Expect(err).ToNot(HaveOccurred())

			// Loop through the image list and use ephemeral containers to inject each image into the test deployment.
			// This will initiate the download of each image into the cluster (if not already present)
			for name, image := range imageList {
				// Use Eventually block because the kubectl commands can get a conflict error when executed so quickly in succession.
				Eventually(func() error {
					return injectImage(kubeconfig, testNamespace, name, image.name, testName)
				}, shortWaitTimeout, pollingInterval).ShouldNot(HaveOccurred(), fmt.Sprintf("failed to inject image %s", image.name))
			}
		})
		t.It(fmt.Sprintf("wait for all ephemeral containers in pod %s to complete", podName), func() {
			Eventually(func() bool {
				return allImagesLoaded(testName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "timed out waiting for images to load")
		})
	})
	metrics.Emit(t.Metrics.With("images_load_elapsed_time", time.Since(start).Milliseconds()))
})

// deployDaemonSet - deploy the DaemonSet for this test
func deployDaemonSet(name string, namespace string) error {
	templateName := "imagepreloader-template.yaml"
	params := map[string]string{
		"test_name":      name,
		"test_namespace": namespace,
	}
	testTemplate, err := template.New(templateName).ParseFiles(fmt.Sprintf("testdata/%s", templateName))
	if err != nil {
		return err
	}

	// Create the Yaml to apply using the template and the substitution parameters
	var buf bytes.Buffer
	err = testTemplate.ExecuteTemplate(&buf, templateName, params)
	if err != nil {
		return err
	}

	// Deploy the Yaml to the cluster
	err = pkg.CreateOrUpdateResourceFromBytes(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

// createImageList - create the list of container images to load into the cluster
func createImageList(bom v8obom.Bom) (map[string]*imageState, error) {
	imageMap := map[string]*imageState{}

	// Process all the images from the BOM
	for _, comp := range bom.GetComponents() {
		for i, subComp := range comp.SubComponents {
			for _, bomImage := range subComp.Images {
				// Special case the platform-operator and application-operator, the images may not exist yet.
				if bomImage.ImageName != "VERRAZZANO_APPLICATION_OPERATOR_IMAGE" &&
					bomImage.ImageName != "VERRAZZANO_PLATFORM_OPERATOR_IMAGE" {
					imageMap[bomImage.ImageName] = &imageState{
						name:   fmt.Sprintf("%s/%s/%s:%s", bom.ResolveRegistry(&comp.SubComponents[i], bomImage), bom.ResolveRepo(&comp.SubComponents[i], bomImage), bomImage.ImageName, bomImage.ImageTag),
						loaded: false,
					}
				}
			}
		}
	}

	// Extract the example images from the example files
	for _, app := range exampleImages {
		imageUrl, err := getExampleImageURL(app)
		Expect(err).ToNot(HaveOccurred())
		imageMap[app.containerName] = &imageState{
			name:   imageUrl,
			loaded: false,
		}
	}

	return imageMap, nil
}

// injectImage - inject a container image into the test pod using ephemeral containers
func injectImage(kubeconfig string, namespace string, containerName string, imageName string, targetName string) error {
	// Override the entrypoint of each image injected to run the "pwd" command.  The container will complete after the command runs.
	cmd := exec.Command("kubectl", "debug", "--namespace", namespace, podName,
		"--container", containerName, "--target", targetName, "--image", imageName, "--image-pull-policy", "IfNotPresent",
		"--", "pwd")
	pkg.Log(pkg.Info, fmt.Sprintf("kubectl command to inject image %s: %s", imageName, cmd.String()))
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

// allImagesLoaded - check if all the images in the ephemeral containers have loaded
func allImagesLoaded(name string, namespace string) bool {
	// Get the pod
	podsList, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"name": name}}, namespace)
	if err != nil {
		return false
	}

	// Loop through the ephemeral containers checking that they are all completed successfully
	pkg.Log(pkg.Info, "Checking if all images are loaded")
	pod := podsList[0]
	for _, container := range pod.Status.EphemeralContainerStatuses {
		// Skip if already marked as loaded
		if !imageList[container.Name].loaded {
			if (container.State.Terminated != nil && container.State.Terminated.Reason == "Completed") ||
				(container.LastTerminationState.Terminated != nil && container.LastTerminationState.Terminated.Reason == "Completed") {
				imageList[container.Name].loaded = true
			} else if container.Name == "coherence-operator" && container.LastTerminationState.Terminated != nil &&
				container.LastTerminationState.Terminated.Reason == "StartError" {
				// Special case coherence-operator - unable to override entrypoint because the image does not contain an OS
				imageList[container.Name].loaded = true
			} else {
				pkg.Log(pkg.Info, fmt.Sprintf("Image %s not loaded yet", container.Image))
			}
		}
	}

	// Determine if all images are loaded
	allImagesLoaded := true
	for _, image := range imageList {
		if !image.loaded {
			allImagesLoaded = false
			break
		}
	}

	if allImagesLoaded {
		pkg.Log(pkg.Info, "All images are loaded")
	}
	return allImagesLoaded
}

// getExampleImageURL - parse the full container image url from the file content
func getExampleImageURL(app exampleApp) (string, error) {
	// Get the content of the file containing the image name
	content, err := readFileToString(app.exampleFile)
	if err != nil {
		return "", err
	}
	// Parse the full image path from the content of the file
	re := regexp.MustCompile(fmt.Sprintf("%s:[a-zA-Z-0-9._]+", app.imageName))
	imageName := re.FindString(content)
	Expect(imageName).ToNot(BeEmpty())
	return imageName, nil
}

// readFileToString - read the contents of a file into a string
func readFileToString(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
