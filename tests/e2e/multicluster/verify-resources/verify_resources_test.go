// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources_test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const shortWaitTimeout = 3 * time.Minute
const shortPollInterval = 5 * time.Second

var t = framework.NewTestFramework("resources_test")

var _ = t.AfterSuite(func() {})
var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Multi Cluster Verify Resources", Label("f:multicluster.register"), func() {
	t.Context("Admin Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})

		t.It("Create VerrazzanoProject with invalid content", func() {
			Eventually(func() bool {
				file, err := pkg.FindTestDataFile("testdata/multicluster/verrazzanoproject-placement-clusters-invalid.yaml")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid VerrazzanoProject: %v", err))
					return false
				}
				err = resource.CreateOrUpdateResourceFromFile(file, t.Logs)
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid VerrazzanoProject")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid VerrazzanoProject: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(BeTrue(), "Expected VerrazzanoProject validation error")
		})

		t.It("Create MultiClusterSecret with invalid content", func() {
			Eventually(func() bool {
				file, err := pkg.FindTestDataFile("testdata/multicluster/multicluster_secret_placement_clusters_invalid.yaml")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				err = resource.CreateOrUpdateResourceFromFile(file, t.Logs)
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(BeTrue(), "Expected a resource validation error")
		})

		t.It("Create MultiClusterConfigmap with invalid content", func() {
			Eventually(func() bool {
				file, err := pkg.FindTestDataFile("testdata/multicluster/multicluster_configmap_placement_clusters_invalid.yaml")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				err = resource.CreateOrUpdateResourceFromFile(file, t.Logs)
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(BeTrue(), "Expected a resource validation error")
		})

		t.It("Create MultiClusterComponent with invalid content", func() {
			Eventually(func() bool {
				file, err := pkg.FindTestDataFile("testdata/multicluster/multicluster_component_placement_clusters_invalid.yaml")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				err = resource.CreateOrUpdateResourceFromFile(file, t.Logs)
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(BeTrue(), "Expected a resource validation error")
		})

		t.It("Create MultiClusterApplicationConfiguration with invalid content", func() {
			Eventually(func() bool {
				file, err := pkg.FindTestDataFile("testdata/multicluster/multicluster_appconf_placement_clusters_invalid.yaml")
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				err = resource.CreateOrUpdateResourceFromFile(file, t.Logs)
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(BeTrue(), "Expected a resource validation error")
		})

	})

})
