// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package noneprofile

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

var allowedNamespaces = []string{
	"default", "kube-system", "kube-node-lease", "kube-public",
	"monitoring", "local-path-storage", "metallb-system",
	"cattle-system", "verrazzano-install", "verrazzano-mc", "verrazzano-system",
	"verrazzano-monitoring", "verrazzano-ingress-nginx",
}

var beforesuite = t.BeforeSuiteFunc(func() {
})

var _ = t.Describe("Verify None Profile Install", func() {
	t.It("Should have the none profile installed and in Ready state", func() {
		// Verify that none profile installation succeeded
		Eventually(func() error {
			return pkg.VzReadyV1beta1()
		}, waitTimeout, pollingInterval).Should(BeNil(), "Expected to get Verrazzano CR with no error")
	})
})

var _ = t.Describe("Verify Namespaces", func() {
	t.It("Should have only certain set of namespaces", func() {
		allowedNamespacesMap := make(map[string]bool)
		for _, item := range allowedNamespaces {
			allowedNamespacesMap[item] = true
		}

		ns, err := pkg.ListNamespaces(metav1.ListOptions{})
		Expect(err).Should(BeNil())

		for _, item := range ns.Items {
			_, isAllowed := allowedNamespacesMap[item.Name]
			if !isAllowed {
				t.Logs.Errorf("Namespace %s is not allowed with none profile installation, Allowed namespaces are %v\n", item.Name, allowedNamespacesMap)
			}
			Expect(isAllowed).To(BeTrue())
		}
	})
})
