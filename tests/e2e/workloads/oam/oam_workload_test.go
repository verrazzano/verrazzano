// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	shortWaitTimeout     = 2 * time.Minute
	shortPollingInterval = 10 * time.Second
	longWaitTimeout      = 5 * time.Minute

	appConfiguration  = "tests/testdata/test-applications/oam/oam-app.yaml"
	compConfiguration = "tests/testdata/test-applications/oam/oam-comp.yaml"

	pvcName = "test-pvc"
)

var (
	t                  = framework.NewTestFramework("oam_workloads")
	generatedNamespace = pkg.GenerateNamespace("oam-workloads")
)

var _ = BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()
		deployOAMApp()
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
	beforeSuitePassed = true
})

var failed = false
var beforeSuitePassed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(namespace)
	}
	if !skipUndeploy {
		undeployOAMApp()
	}
})

var _ = t.Describe("An OAM application is deployed", Label("f:app-lcm.oam"), func() {
	t.Context("Check for created resources", func() {
		// GIVEN the test application is deployed
		// AND the application includes a component that wraps a persistent volume claim
		// WHEN a call is made to fetch the persistent volume claim
		// THEN the persistent volume claim is found to exist
		t.It("The persistent volume claim exists", func() {
			Eventually(func() (bool, error) {
				return volumeClaimExists(pvcName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
		})
	})
})

// volumeClaimExists checks if the persistent volume claim with the specified name exists
// in the test namespace
func volumeClaimExists(volumeClaimName string) (bool, error) {
	volumeClaims, err := pkg.GetPersistentVolumeClaims(namespace)
	if err != nil {
		return false, err
	}
	_, ok := volumeClaims[volumeClaimName]
	return ok, nil
}

// deployOAMApp deploys the test components and application configuration
func deployOAMApp() {
	t.Logs.Info("Deploy OAM application")

	t.Logs.Info("Create namespace")
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true"}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(compConfiguration)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval, "Failed to create component resources for Coherence application").ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resources")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(appConfiguration)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())
}

// undeployOAMApp removes the test components and application configuration
func undeployOAMApp() {
	t.Logs.Info("Undeploy OAM application")
	start := time.Now()
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(appConfiguration)
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(compConfiguration)
		if err != nil {
			return err
		}
		return resource.DeleteResourceFromFileInGeneratedNamespace(file, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for persistent volume claim to terminate")
	Eventually(func() (bool, error) {
		return volumeClaimExists(pvcName)
	}, shortWaitTimeout, shortPollingInterval).Should(BeFalse())

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for namespace finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Wait for namespace deletion")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, longWaitTimeout, shortPollingInterval).Should(BeTrue())

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}
