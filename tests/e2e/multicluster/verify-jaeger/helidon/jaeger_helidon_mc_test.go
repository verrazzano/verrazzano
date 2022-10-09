// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	projectName          = "hello-helidon-jaeger"
)

const (
	testAppComponentFilePath     = "testdata/jaeger/helidon/multicluster/mc-helidon-tracing-comp.yaml"
	testAppConfigurationFilePath = "testdata/jaeger/helidon/multicluster/mc-helidon-tracing-app.yaml"
	verrazzanoProjectFilePath    = "testdata/jaeger/helidon/multicluster/helidon-verrazzano-project.yaml"
)

var (
	t                        = framework.NewTestFramework("jaeger-mc-helidon")
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	beforeSuitePassed        = false
	failed                   = false
	start                    = time.Now()
	helloHelidonServiceName  = "hello-helidon-jaeger-mc"
)

var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = t.BeforeSuite(func() {
	start = time.Now()
	// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources

	if adminKubeconfig == "" || managedKubeconfig == "" || managedClusterName == "" {
		AbortSuite("One or more required env variables (ADMIN_KUBECONFIG, MANAGED_KUBECONFIG, MANAGED_CLUSTER_NAME) for the test suite are not set.")
	}
	// deploy the VerrazzanoProject
	start := time.Now()
	Eventually(func() error {
		if err := pkg.CreateOrUpdateResourceFromFileInCluster(verrazzanoProjectFilePath, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to create %s project resource: %v", projectName, err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		_, err := pkg.GetNamespaceInCluster(projectName, adminKubeconfig)
		return err == nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())

	Eventually(func() error {
		if err := pkg.CreateOrUpdateResourceFromFileInCluster(testAppComponentFilePath, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to create multi-cluster %s component resources: %v", projectName, err)
		}
		if err := pkg.CreateOrUpdateResourceFromFileInCluster(testAppConfigurationFilePath, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to create multi-cluster %s application resource: %v", projectName, err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())
	Eventually(func() bool {
		result, err := pkg.PodsRunningInCluster(projectName, expectedPodsHelloHelidon, managedKubeconfig)
		if err != nil {
			return false
		}
		return result
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	err := pkg.GenerateTrafficForTraces(projectName, "", "greet", managedKubeconfig)
	if err != nil {
		pkg.Log(pkg.Error, "Unable to send traffic requests to generate traces")
	}
	beforeSuitePassed = true
})

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		err := pkg.ExecuteBugReport(projectName)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
		}
	}
	// undeploy the application here
	start := time.Now()
	Eventually(func() error {
		if err := pkg.DeleteResourceFromFileInCluster(testAppConfigurationFilePath, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to delete multi-cluster hello-helidon application resource: %v", err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())
	Eventually(func() error {
		if err := pkg.DeleteResourceFromFileInCluster(testAppComponentFilePath, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to delete multi-cluster hello-helidon component resources: %v", err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	Eventually(func() error {
		if err := pkg.DeleteResourceFromFileInCluster(verrazzanoProjectFilePath, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to delete hello-helidon project resource: %v", err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(projectName, managedKubeconfig)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())
	Eventually(func() error {
		return pkg.DeleteNamespaceInCluster(projectName, adminKubeconfig)
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Helidon App with Jaeger Traces", Label("f:jaeger.helidon-workload"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is enabled and a sample application is installed,
		// WHEN we check for traces for that service,
		// THEN we are able to get the traces
		t.It("traces for the helidon app should be available when queried from Jaeger", func() {
			validatorFn := pkg.ValidateApplicationTracesInCluster(adminKubeconfig, start, helloHelidonServiceName, managedClusterName)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

	})
})
