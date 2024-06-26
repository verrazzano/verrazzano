// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifynoneprofile

import (
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var t = framework.NewTestFramework("none-profile")

var _ = t.BeforeSuiteFunc(beforesuite)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

// allowedNamespaces holds the list of namespaces permissible for installation with the default 'none' profile.
// Any other namespace not mentioned here is considered unnecessary
var allowedNamespaces = []string{
	"default", "kube-system", "kube-node-lease", "kube-public",
	"cattle-system", "monitoring", "local-path-storage", "metallb-system",
	"verrazzano-install", "verrazzano-system",
}

var beforesuite = t.BeforeSuiteFunc(func() {
})

var _ = t.Describe("Verify None Profile Install", func() {
	t.It("Should have the none profile installed and in Ready state", func() {
		// Verify that none profile installation succeeded
		Eventually(func() (bool, error) {
			isReady, err := pkg.VzReadyV1beta1()
			return isReady, err
		}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected Verrazzano CR to be in Ready state")
	})
})

var _ = t.Describe("Verify Namespaces", func() {
	t.It("Should have only certain set of namespaces", func() {
		allowedNamespacesMap := make(map[string]bool)
		for _, item := range allowedNamespaces {
			allowedNamespacesMap[item] = true
		}

		ns, err := pkg.ListNamespaces(metav1.ListOptions{})
		Expect(err).ShouldNot(HaveOccurred())

		failures := []string{}
		for _, item := range ns.Items {
			_, isAllowed := allowedNamespacesMap[item.Name]
			if !isAllowed {
				failures = append(failures, item.Name)
			}
		}
		Expect(failures).To(BeEmpty(), "Namespaces not allowed: %v, Allowed namespaces are only %v", failures, allowedNamespaces)
	})
})
