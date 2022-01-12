// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagepreloader

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

const testNamespace string = "image-preloader"
const testName string = "preloader-test"
const bomPath string = "../../../platform-operator/verrazzano-bom.json"

var bom v8obom.Bom
var imageList map[string]string
var podName string
var kubeconfig string

var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortWaitTimeout = 5 * time.Minute
var shortPollingInterval = 10 * time.Second

var t = framework.NewTestFramework("imagepreloader")

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
})

var _ = t.Describe("Load Verrazzano Container Images", func() {
	start := time.Now()
	t.Context("Deployment", func() {
		t.It("deploy test pod", func() {
			err := deployDaemonSet()
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
					return injectImage(kubeconfig, name, image)
				}, shortWaitTimeout, pollingInterval).ShouldNot(HaveOccurred(), fmt.Sprintf("failed to inject image %s", image))
			}
		})
		t.It(fmt.Sprintf("wait for all ephemeral containers in pod %s to complete", podName), func() {
			Eventually(func() bool {
				return areImagesToLoaded(testName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "timed out waiting for images to load")
		})
	})
	metrics.Emit(t.Metrics.With("images_load_elapsed_time", time.Since(start).Milliseconds()))
})

// deployDaemonSet - deploy the DaemonSet for this test
func deployDaemonSet() error {
	templateName := "imagepreloader-template.yaml"
	params := map[string]string{
		"test_name":      testName,
		"test_namespace": testNamespace,
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
func createImageList(bom v8obom.Bom) (map[string]string, error) {
	imageMap := map[string]string{}
	for _, comp := range bom.GetComponents() {
		for i, subComp := range comp.SubComponents {
			for _, bomImage := range subComp.Images {
				// Special case the platform-operator and application-operator, the images may not exist yet.
				// Special case coherence-operator - unable to override entrypoint with a simple command
				if bomImage.ImageName != "VERRAZZANO_APPLICATION_OPERATOR_IMAGE" &&
					bomImage.ImageName != "VERRAZZANO_PLATFORM_OPERATOR_IMAGE" &&
					bomImage.ImageName != "coherence-operator" {
					imageMap[bomImage.ImageName] = fmt.Sprintf("%s/%s/%s:%s", bom.ResolveRegistry(&comp.SubComponents[i], bomImage), bom.ResolveRepo(&comp.SubComponents[i], bomImage), bomImage.ImageName, bomImage.ImageTag)
				}
			}
		}
	}
	return imageMap, nil
}

// injectImage - inject a container image into the test pod using ephemeral containers
func injectImage(kubeconfig string, name string, image string) error {
	cmd := exec.Command("kubectl", "debug", "--namespace", testNamespace, podName,
		"--container", name, "--target", testName, "--image", image, "--image-pull-policy", "IfNotPresent",
		"--", "pwd")
	pkg.Log(pkg.Info, fmt.Sprintf("kubectl command to inject image %s: %s", image, cmd.String()))
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

// areImagesToLoaded - wait for the images in the ephemeral containers to load
func areImagesToLoaded(name string, namespace string) bool {
	// Get the pod
	podsList, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"name": name}}, namespace)
	if err != nil {
		return false
	}

	// Loop through the ephemeral containers checking that they are all completed successfully
	pod := podsList[0]
	allImagesLoaded := true
	for _, container := range pod.Status.EphemeralContainerStatuses {
		if container.State.Terminated == nil || container.State.Terminated.Reason != "Completed" {
			allImagesLoaded = false
			break
		}
	}
	return allImagesLoaded
}
