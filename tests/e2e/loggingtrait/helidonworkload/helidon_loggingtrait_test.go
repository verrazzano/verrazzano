// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonworkload

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/loggingtrait"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	componentsPath       = "testdata/loggingtrait/helidonworkload/helidon-logging-components.yaml"
	applicationPath      = "testdata/loggingtrait/helidonworkload/helidon-logging-application.yaml"
	applicationPodName   = "hello-helidon-deployment-"
	configMapName        = "logging-stdout-hello-helidon-deployment-deployment"
)

var (
	kubeConfig         = os.Getenv("KUBECONFIG")
	t                  = framework.NewTestFramework("helidonworkload")
	generatedNamespace = pkg.GenerateNamespace("hello-helidon-logging")
	clusterDump        = pkg.NewClusterDumpWrapper(generatedNamespace)
)

var _ = clusterDump.BeforeSuite(func() {
	loggingtrait.DeployApplication(namespace, istioInjection, componentsPath, applicationPath, applicationPodName, t)
})

var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	loggingtrait.UndeployApplication(namespace, componentsPath, applicationPath, configMapName, t)
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("Test helidon loggingtrait application", Label("f:app-lcm.oam",
	"f:app-lcm.helidon-workload",
	"f:app-lcm.logging-trait"), func() {

	t.Context("for LoggingTrait.", func() {
		// GIVEN the app is deployed and the pods are running
		// WHEN the app pod is inspected
		// THEN the container for the logging trait should exist
		t.It("Verify that 'logging-stdout' container exists in the 'hello-helidon-deployment' pod", func() {
			Eventually(func() bool {
				containerExists, err := pkg.DoesLoggingSidecarExist(kubeConfig, types.NamespacedName{Name: applicationPodName, Namespace: namespace}, "logging-stdout")
				return containerExists && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})

		// GIVEN the app is deployed and the pods are running
		// WHEN the configmaps in the app namespace are retrieved
		// THEN the configmap for the logging trait should exist
		t.It("Verify that 'logging-stdout-hello-helidon-deployment-deployment' ConfigMap exists in the 'hello-helidon-logging' namespace", func() {
			Eventually(func() bool {
				configMap, err := pkg.GetConfigMap(configMapName, namespace)
				return (configMap != nil) && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})
})
