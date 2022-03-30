// Copyright (C) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/platform-operator/test/integ/k8s"
	"github.com/verrazzano/verrazzano/platform-operator/test/integ/util"
)

const clusterAdmin = "cluster-admin"
const platformOperator = "verrazzano-platform-operator"
const installNamespace = "verrazzano-install"

const vzResourceNamespace = "default"
const vzResourceName = "test"

var K8sClient k8s.Client

var _ = BeforeSuite(func() {
	var err error
	K8sClient, err = k8s.NewClient(util.GetKubeconfig())
	if err != nil {
		Fail(fmt.Sprintf("Error creating Kubernetes client to access Verrazzano API objects: %v", err))
	}

	// Platform operator pod is eventually running
	isPodRunningYet := func() bool {
		return K8sClient.IsPodRunning(platformOperator, installNamespace)
	}
	Eventually(isPodRunningYet, "2m", "5s").Should(BeTrue(),
		"The verrazzano-platform-operator pod should be in the Running state")

	// Create multi-cluster namespace
	if !K8sClient.DoesNamespaceExist(constants.VerrazzanoMultiClusterNamespace) {
		err = K8sClient.EnsureNamespace(constants.VerrazzanoMultiClusterNamespace)
		Expect(err).To(BeNil())
	}

	// Create verrazzano-system namespace
	if !K8sClient.DoesNamespaceExist(constants.VerrazzanoSystemNamespace) {
		err = K8sClient.EnsureNamespace(constants.VerrazzanoSystemNamespace)
		Expect(err).To(BeNil())
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
	It("Platform operator pod is eventually running", func() {
		isPodRunningYet := func() bool {
			return K8sClient.IsPodRunning(platformOperator, installNamespace)
		}
		Eventually(isPodRunningYet, "2m", "5s").Should(BeTrue(),
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

var _ = Describe("Custom Resource Definition for Verrazzano install", func() {
	It("verrazzanos.install.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanos.install.verrazzano.io")).To(BeTrue(),
			"The verrazzanos.install.verrazzano.io CRD should exist")
	})
	It("verrazzanomanagedclusters.clusters.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanomanagedclusters.clusters.verrazzano.io")).To(BeTrue(),
			"The verrazzanomanagedclusters.clusters.verrazzano.io CRD should exist")
	})
})

var _ = Describe("Install with enable/disable component", func() {
	It("CRD verrazzanos.install.verrazzano.io exists", func() {
		Expect(K8sClient.DoesCRDExist("verrazzanos.install.verrazzano.io")).To(BeTrue(),
			"The verrazzanos.install.verrazzano.io CRD should exist")
	})

	It("Platform operator pod is eventually running", func() {
		isPodRunningYet := func() bool {
			return K8sClient.IsPodRunning(platformOperator, installNamespace)
		}
		Eventually(isPodRunningYet, "2m", "5s").Should(BeTrue(),
			"The verrazzano-platform-operator pod should be in the Running state")
	})

	It("Verrazzano CR should have disabled components", func() {
		_, stderr := util.Kubectl("apply -f testdata/install-disabled.yaml")
		Expect(stderr).To(Equal(""))

		Eventually(func() bool {
			return checkAllComponentStates(vzapi.CompStateDisabled)
		}, "10s", "1s").Should(BeTrue())
	})
	It("Verrazzano CR should have preInstalling or installing components", func() {
		_, stderr := util.Kubectl("apply -f testdata/install-enabled.yaml")
		Expect(stderr).To(Equal(""))

		Eventually(func() bool {
			return checkAllComponentStates(vzapi.CompStatePreInstalling, vzapi.CompStateInstalling)

		}, "30s", "1s").Should(BeTrue())
	})
})

// Check if Verrazzano CR has one matching state all components being tested
func checkAllComponentStates(states ...vzapi.CompStateType) bool {
	if !checkStates(coherence.ComponentName, states...) {
		return false
	}
	if !checkStates(weblogic.ComponentName, states...) {
		return false
	}
	return true
}

// Check if Verrazzano CR has one matching state for specified component
func checkStates(compName string, states ...vzapi.CompStateType) bool {
	vzcr, err := K8sClient.GetVerrazzano(vzResourceNamespace, vzResourceName)
	if err != nil {
		return false
	}
	if vzcr.Status.Components == nil {
		return false
	}
	// Check if the component matches one of the states
	for _, comp := range vzcr.Status.Components {
		if comp.Name == compName {
			for _, state := range states {
				if comp.State == state {
					return true
				}
			}
		}
	}
	return false
}
