// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	v1 "k8s.io/api/core/v1"
	"time"
)

const (
	loggingNamespace = "verrazzano-logging"
	verrazzanoName   = "verrazzano"
	timeout          = 15 * time.Minute
	pollInterval     = 10 * time.Second
)

var (
	t = framework.NewTestFramework("install")
	//client *vpoClient.Clientset
	//restConfig    *rest.Config
)

var beforeSuitePassed = false
var _ = t.AfterEach(func() {
	CurrentSpecReport().Failed()
})

var afterSuite = t.AfterSuiteFunc(func() {
	pkg.UninstallOpenSearchOperator()
})

var _ = AfterSuite(afterSuite)
var _ = BeforeSuite(beforeSuite)

var beforeSuite = t.BeforeSuiteFunc(func() {
	t.Logs.Info(fmt.Sprintf("Creating %s namespace", loggingNamespace))
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
		}
		return pkg.CreateNamespace(loggingNamespace, nsLabels)
	}, timeout, pollInterval).ShouldNot(BeNil())

	t.Logs.Info("Install opensearch-operator and cluster")
	Expect(func() error {
		err := pkg.InstallOpenSearchOperator(t.Logs)
		return err
	}).NotTo(HaveOccurred())

	beforeSuitePassed = true
})

var _ = t.Describe("OpenSearch Verify Install", func() {
	t.It("sts are ready", func() {
		Eventually(func() bool {
			isReady, err := pkg.PodsRunning(loggingNamespace, []string{"opensearch"})
			if err != nil {
				return false
			}
			return isReady
		}, timeout, pollInterval).Should(BeTrue(), "OpenSearch failed to get to ready state")
	})
})
