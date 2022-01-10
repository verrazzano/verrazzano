// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package imagepreloader

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v8obom "github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
)

const namespace string = "image-preloader"
const testName string = "preloader-test"
const bomPath string = "../../../platform-operator/verrazzano-bom.json"

var bom v8obom.Bom
var waitTimeout = 10 * time.Minute
var pollingInterval = 30 * time.Second
var shortWaitTimeout = 5 * time.Minute
var shortPollingInterval = 10 * time.Second

var _ = BeforeSuite(func() {
	pkg.Log(pkg.Info, "Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		return pkg.CreateNamespace(namespace, map[string]string{"verrazzano-managed": "true", "istio-injection": "enabled"})
	}, waitTimeout, shortPollingInterval).ShouldNot(BeNil())

	pkg.Log(pkg.Info, "Parse Verrazzano bom file")
	var err error
	bom, err = v8obom.NewBom(bomPath)
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	pkg.Log(pkg.Info, "Delete the namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, waitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
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
