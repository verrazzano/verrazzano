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
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const namespace string = "image-preloader"
const testName string = "preloader-test"
const bomPath string = "../../../platform-operator/verrazzano-bom.json"

var bom v8obom.Bom
var imageList map[string]string
var podName string

var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortWaitTimeout = 5 * time.Minute
var shortPollingInterval = 10 * time.Second

var _ = BeforeSuite(func() {
	pkg.Log(pkg.Info, fmt.Sprintf("Create namespace %s", namespace))
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(namespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Parse Verrazzano bom file")
	var err error
	bom, err = v8obom.NewBom(bomPath)
	Expect(err).ToNot(HaveOccurred())
	pkg.Log(pkg.Info, fmt.Sprintf("The Bom version is %s", bom.GetVersion()))
})

var _ = AfterSuite(func() {
	pkg.Log(pkg.Info, fmt.Sprintf("Delete the namespace %s", namespace))
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	pkg.Log(pkg.Info, fmt.Sprintf("Wait for the namespace %s to be terminated", namespace))
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
})

var _ = Describe("Load Verrazzano Container Images", func() {
	Context("Deployment", func() {
		It("deploy test pod", func() {
			err := deployDaemonSet()
			Expect(err).ToNot(HaveOccurred())
		})
		It("wait for the pod to be ready", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{testName})
			}, shortWaitTimeout, pollingInterval).Should(BeTrue(), fmt.Sprintf("%s failed to deploy", testName))
		})
		It("get the pod name", func() {
			var podsList []corev1.Pod
			Eventually(func() error {
				var err error
				podsList, err = pkg.GetPodsFromSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"name": testName}}, namespace)
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
	Context("Use ephemeral containers to inject images", func() {
		It("create container list", func() {
			var err error
			imageList, err = createImageList(bom)
			Expect(err).ToNot(HaveOccurred())
		})
		It(fmt.Sprintf("inject images into the test pod %s", podName), func() {
			// Get the kubeconfig location
			kubeconfig, err := k8sutil.GetKubeConfigLocation()
			Expect(err).ToNot(HaveOccurred())
			Expect(kubeconfig).ToNot(HaveLen(0))
			err = injectImages(kubeconfig, imageList)
			Expect(err).ToNot(HaveOccurred())

		})
	})
})

// deployDaemonSet - deploy the DaemonSet for this test
func deployDaemonSet() error {
	templateName := "imagepreloader-template.yaml"
	params := map[string]string{
		"test_name":      testName,
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
func createImageList(bom v8obom.Bom) (map[string]string, error) {
	imageMap := map[string]string{}
	for _, comp := range bom.GetComponents() {
		for _, subComp := range comp.SubComponents {
			for _, bomImage := range subComp.Images {
				// Special case the platform-operator and application-operator, the images may not exist yet
				if bomImage.ImageName != "VERRAZZANO_APPLICATION_OPERATOR_IMAGE" && bomImage.ImageName != "VERRAZZANO_PLATFORM_OPERATOR_IMAGE" {
					imageMap[bomImage.ImageName] = fmt.Sprintf("%s/%s/%s:%s", bom.ResolveRegistry(&subComp, bomImage), bom.ResolveRepo(&subComp, bomImage), bomImage.ImageName, bomImage.ImageTag)
				}
			}
		}
	}
	return imageMap, nil
}

// injectImages - inject the container images into the test pod using ephemeral containers
func injectImages(kubeconfig string, imageList map[string]string) error {
	// Loop through the image list and use ephemeral containers to inject each image into the test deployment.
	// This will initiate the download of each image into the cluster (if not already present)
	for name, image := range imageList {
		cmd := exec.Command("kubectl", "debug", "--namespace", namespace, podName,
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
	}
	return nil
}
