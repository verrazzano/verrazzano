// Copyright (C) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/operator/test/integ/k8s"
)

const clusterAdmin = "cluster-admin"
const platformOperator = "verrazzano-platform-operator"
const installNamespace = "verrazzano-install"

var K8sClient k8s.Client

var _ = BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient()
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}
})

var _ = AfterSuite(func() {
})

var _ = Describe("verrazzano-install namespace resources ", func() {
	It(fmt.Sprintf("Namespace %s exists", installNamespace), func() {
		Expect(K8sClient.DoesNamespaceExist(installNamespace)).To(BeTrue(),
			"The install-namespace should exist")
	})
	It(fmt.Sprintf("ServiceAccount %s exists", platformOperator), func() {
		Expect(K8sClient.DoesServiceAccountExist(platformOperator, installNamespace)).To(BeTrue(),
			"The verrazzano-platform-operator service should exist")
	})
	It(fmt.Sprintf("Deployment %s exists", platformOperator), func() {
		Expect(K8sClient.DoesDeploymentExist(platformOperator, installNamespace)).To(BeTrue(),
			"The verrazzano-platform-operator should exist")
	})
	It(fmt.Sprintf("Pod prefixed by %s exists", platformOperator), func() {
		Expect(K8sClient.DoesPodExist(platformOperator, installNamespace)).To(BeTrue(),
			"The verrazzano-platform-operator pod should exist")
	})
	It("is running (within 1m)", func() {
		isPodRunningYet := func() bool {
			return K8sClient.IsPodRunning(platformOperator, installNamespace)
		}
		Eventually(isPodRunningYet, "1m", "5s").Should(BeTrue(),
			"The verrazzano-platform-operator pod should be in the Running state")
	})
})

var _ = Describe("Verrazzano cluster roles and bindings for platform operator", func() {
	It(fmt.Sprintf("Cluster admin role %s exists", clusterAdmin), func() {
		Expect(K8sClient.DoesClusterRoleExist(clusterAdmin)).To(BeTrue(),
			"The cluster-admin role should exist")
	})
	It(fmt.Sprintf("Cluster role binding for platform operator %s exists", platformOperator), func() {
		Expect(K8sClient.DoesClusterRoleBindingExist(platformOperator)).To(BeTrue(),
			"The cluster role binding for verrazzano-platform-operator should exist")
	})

})

var _ = Describe("Custom Resource Definition for verrazzano install", func() {
	It("verrazzanos.install.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanos.install.verrazzano.io")).To(BeTrue(),
			"The verrazzanos.install.verrazzano.io CRD should exist")
	})
})
