// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package resources_test

import (
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const shortWaitTimeout = 10 * time.Minute
const shortPollInterval = 10 * time.Second

const multiclusterNamespace = "verrazzano-mc"
const verrazzanoSystemNamespace = "verrazzano-system"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = ginkgo.Describe("Multi Cluster Verify Resources", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("Create VerrazzanoProject with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/verrazzanoproject-placement-cluster-invalid.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid VerrazzanoProject")
					return false
				}
				if strings.Contains(err.Error(), "admission webhook") {
					return true
				}
				return false
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected VerrazzanoProject validation error")
		})

	})

	ginkgo.Context("Managed Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("Create VerrazzanoProject with invalid content", func() {
			gomega.Eventually(func() bool {
				err := pkg.CreateOrUpdateResourceFromFile("testdata/multicluster/verrazzanoproject-placement-clusters-empty.yaml")
				if err == nil {
					pkg.Log(pkg.Error, "Expected an error creating invalid VerrazzanoProject")
					return false
				}
				if strings.Contains(err.Error(), "admission webhook") {
					return true
				}
				return false
			}, shortWaitTimeout, shortPollInterval).Should(gomega.BeTrue(), "Expected VerrazzanoProject validation error")
		})
	})
})
