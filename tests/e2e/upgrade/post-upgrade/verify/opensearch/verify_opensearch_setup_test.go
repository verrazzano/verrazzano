// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("opensearch")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Post upgrade", func() {
	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f interface{}) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Expect(err).To(BeNil(), fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
		}
		supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
		if err != nil {
			Fail(err.Error())
		}
		// Only run tests if Verrazzano is at least version 1.3.0
		if supported {
			t.It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.3.0", description))
		}
	}

	// GIVEN the OpenSearch pod
	// WHEN the indices are retrieved
	// THEN verify that they do not have the old indices
	MinimumVerrazzanoIt("Old indices are deleted", func() {
		Eventually(func() bool {
			kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
			enabled, err := pkg.IsOpenSearchEnabled(kubeconfigPath)
			if err != nil {
				return false
			}
			if enabled {
				oldIndicesPatterns := []string{"^verrazzano-namespace-.*$", "^verrazzano-systemd-journal$",
					"^verrazzano-logstash-.*$"}
				return pkg.IndicesNotExists(oldIndicesPatterns)
			}
			return true
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected not to find any old indices")
	})
})
