// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherenceworkload

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/loggingtrait"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "sockshop-logging"
	componentsPath       = "testdata/loggingtrait/coherenceworkload/coherence-logging-components.yaml"
	applicationPath      = "testdata/loggingtrait/coherenceworkload/coherence-logging-application.yaml"
	applicationPodName   = "carts-coh-0"
	configMapName        = "logging-stdout-carts-coh-coherence"
)

var kubeConfig = os.Getenv("KUBECONFIG")

var t = framework.NewTestFramework("coherenceworkload")

var _ = t.BeforeSuite(func() {
	loggingtrait.DeployApplication(namespace, componentsPath, applicationPath, t)
})

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	loggingtrait.UndeployApplication(namespace, componentsPath, applicationPath, configMapName, t)
})

var _ = t.Describe("Test coherence loggingtrait application", func() {

	t.Context("Deployment.", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and MySQL pods should be found running
		t.It("Verify 'hello-helidon-deployment' pod is running", func() {
			t.Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{applicationPodName})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	t.Context("for LoggingTrait.", func() {
		// GIVEN the app is deployed and the pods are running
		// WHEN the app pod is inspected
		// THEN the container for the logging trait should exist
		t.It("Verify that 'logging-stdout' container exists in the 'carts-coh-0' pod", func() {
			t.Eventually(func() bool {
				containerExists, err := pkg.DoesLoggingSidecarExist(kubeConfig, types.NamespacedName{Name: applicationPodName, Namespace: namespace}, "logging-stdout")
				return containerExists && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})

		// GIVEN the app is deployed and the pods are running
		// WHEN the configmaps in the app namespace are retrieved
		// THEN the configmap for the logging trait should exist
		t.It("Verify that 'logging-stdout-carts-coh-coherence' ConfigMap exists in the 'sockshop-logging' namespace", func() {
			t.Eventually(func() bool {
				configMap, err := pkg.GetConfigMap(configMapName, namespace)
				return (configMap != nil) && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})
})
