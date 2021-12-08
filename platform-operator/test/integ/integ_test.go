// Copyright (C) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package integ_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

})

var _ = AfterSuite(func() {
})


var _ = Describe("Install with enable/disable component", func() {

	It("Verrazzano CR should have disabled components", func() {
		_, stderr := util.Kubectl("apply -f testdata/install-disabled.yaml")
		Expect(stderr).To(Equal(""))

		Eventually(func() bool {
			return checkAllComponentStates(vzapi.Disabled)
		}, "10s", "1s").Should(BeTrue())
	})
	It("Verrazzano CR should have preInstalling or installing components", func() {
		_, stderr := util.Kubectl("apply -f testdata/install-enabled.yaml")
		Expect(stderr).To(Equal(""))

		Eventually(func() bool {
			return checkAllComponentStates(vzapi.PreInstalling, vzapi.Installing)

		}, "30s", "1s").Should(BeTrue())
	})
})

// Check if Verrazzano CR has one matching state all components being tested
func checkAllComponentStates(states ...vzapi.StateType) bool {
	if !checkStates(coherence.ComponentName, states...) {
		return false
	}
	if !checkStates(weblogic.ComponentName, states...) {
		return false
	}
	return true
}

// Check if Verrazzano CR has one matching state for specified component
func checkStates(compName string, states ...vzapi.StateType) bool {
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
