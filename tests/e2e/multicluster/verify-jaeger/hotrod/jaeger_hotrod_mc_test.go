// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package hotrod

import (
	"fmt"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
	projectName          = "hotrod"
)

const (
	testAppComponentFilePath     = "testdata/jaeger/hotrod/multicluster/hotrod-tracing-comp.yaml"
	testAppConfigurationFilePath = "testdata/jaeger/hotrod/multicluster/mc-hotrod-tracing-app.yaml"
	verrazzanoProjectFilePath    = "testdata/jaeger/hotrod/multicluster/hotrod-verrazzano-project.yaml"
)

var (
	t                  = framework.NewTestFramework("jaeger-mc-hotrod")
	expectedPodsHotrod = []string{"hotrod-workload"}
	beforeSuitePassed  = false
	failed             = false
	start              = time.Now()
	hotrodServiceName  = "hotrod.hotrod"
)

var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = t.BeforeSuite(func() {
	start = time.Now()
	// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources

	// deploy the VerrazzanoProject
	start := time.Now()
	if adminKubeconfig == "" || managedKubeconfig == "" || managedClusterName == "" {
		AbortSuite("One or more required env variables (ADMIN_KUBECONFIG, MANAGED_KUBECONFIG, MANAGED_CLUSTER_NAME) for the test suite are not set.")
	}
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(verrazzanoProjectFilePath)
		if err != nil {
			return err
		}
		if err := resource.CreateOrUpdateResourceFromFileInCluster(file, adminKubeconfig); err != nil {
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
		file, err := pkg.FindTestDataFile(testAppComponentFilePath)
		if err != nil {
			return err
		}
		if err := resource.CreateOrUpdateResourceFromFileInCluster(file, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to create multi-cluster %s component resources: %v", projectName, err)
		}
		file, err = pkg.FindTestDataFile(testAppConfigurationFilePath)
		if err != nil {
			return err
		}
		if err := resource.CreateOrUpdateResourceFromFileInCluster(file, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to create multi-cluster %s application resource: %v", projectName, err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())
	Eventually(func() bool {
		result, err := pkg.PodsRunningInCluster(projectName, expectedPodsHotrod, managedKubeconfig)
		if err != nil {
			return false
		}
		return result
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	err := pkg.GenerateTrafficForTraces(projectName, "", "dispatch?customer=123", managedKubeconfig)
	if err != nil {
		AbortSuite("Unable to send traffic requests to generate traces")
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
		file, err := pkg.FindTestDataFile(testAppConfigurationFilePath)
		if err != nil {
			return err
		}
		if err := resource.DeleteResourceFromFileInCluster(file, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to delete multi-cluster hotrod application resource: %v", err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(testAppComponentFilePath)
		if err != nil {
			return err
		}
		if err := resource.DeleteResourceFromFileInCluster(file, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to delete multi-cluster hotrod component resources: %v", err)
		}
		return nil
	}).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).ShouldNot(HaveOccurred())

	Eventually(func() error {
		file, err := pkg.FindTestDataFile(verrazzanoProjectFilePath)
		if err != nil {
			return err
		}
		if err := resource.DeleteResourceFromFileInCluster(file, adminKubeconfig); err != nil {
			return fmt.Errorf("failed to delete hotrod project resource: %v", err)
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

var _ = t.Describe("Hotrod App with Jaeger Traces", Label("f:jaeger.hotrod-workload"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is enabled and a sample application is installed,
		// WHEN we check for traces for that service,
		// THEN we are able to get the traces
		t.It("traces for the hotrod app should be available when queried from Jaeger", func() {
			validatorFn := pkg.ValidateApplicationTracesInCluster(adminKubeconfig, start, hotrodServiceName, managedClusterName)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

	})
})
