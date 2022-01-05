// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonlogging

import (
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
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "hello-helidon-logging"
	componentsPath       = "testdata/loggingtrait/helidonworkload/helidon-logging-components.yaml"
	applicationPath      = "testdata/loggingtrait/helidonworkload/helidon-logging-application.yaml"
	applicationPodName   = "hello-helidon-deployment-"
	configMapName        = "logging-stdout-hello-helidon-deployment-deployment"
)

var kubeConfig = os.Getenv("KUBECONFIG")

var _ = BeforeSuite(func() {
	loggingtrait.DeployApplication(namespace, componentsPath, applicationPath)
})

var clusterDump = pkg.NewClusterDumpWrapper()
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	loggingtrait.UndeployApplication(namespace, componentsPath, applicationPath, configMapName)
})

var _ = Describe("Verify application.", func() {

	Context("Deployment.", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the adminserver and mysql pods should be found running
		It("Verify 'hello-helidon-deployment' pod is running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{applicationPodName})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	Context("LoggingTrait.", func() {
		// GIVEN the app is deployed and the pods are running
		// WHEN the app pod is inspected
		// THEN the container for the logging trait should exist
		It("Verify that 'logging-stdout' container exists in the 'hello-helidon-deployment' pod", func() {
			Eventually(func() bool {
				containerExists, err := pkg.DoesLoggingSidecarExist(kubeConfig, types.NamespacedName{Name: applicationPodName, Namespace: namespace}, "logging-stdout")
				return containerExists && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})

		// GIVEN the app is deployed and the pods are running
		// WHEN the configmaps in the app namespace are retrieved
		// THEN the configmap for the logging trait should exist
		It("Verify that 'logging-stdout-hello-helidon-deployment-deployment' ConfigMap exists in the 'hello-helidon-logging' namespace", func() {
			Eventually(func() bool {
				configMap, err := pkg.GetConfigMap(configMapName, namespace)
				return (configMap != nil) && (err == nil)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})
})
