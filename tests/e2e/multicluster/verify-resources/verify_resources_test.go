// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources_test

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const shortWaitTimeout = 3 * time.Minute
const shortPollInterval = 5 * time.Second

var _ = ginkgo.Describe("Multi Cluster Verify Resources", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("Create VerrazzanoProject with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/verrazzanoproject-placement-clusters-invalid.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid VerrazzanoProject")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid VerrazzanoProject: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected VerrazzanoProject validation error")
		})

		ginkgo.It("Create MultiClusterSecret with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_secret_placement_clusters_invalid.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected a resource validation error")
		})

		ginkgo.It("Create MultiClusterConfigmap with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_configmap_placement_clusters_invalid.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected a resource validation error")
		})

		ginkgo.It("Create MultiClusterComponent with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_component_placement_clusters_invalid.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected a resource validation error")
		})

		ginkgo.It("Create MultiClusterApplicationConfiguration with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/multicluster_appconf_placement_clusters_invalid.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid resource")
					return false
				}
				if !strings.Contains(err.Error(), "invalid-cluster-name") {
					pkg.Log(pkg.Error, fmt.Sprintf("Expected an error message creating invalid resource: %v", err))
					return false
				}
				return true
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected a resource validation error")
		})

	})

})
