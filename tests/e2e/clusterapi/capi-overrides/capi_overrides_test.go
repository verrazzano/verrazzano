// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi_overrides

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	capipkg "github.com/verrazzano/verrazzano/tests/e2e/pkg/clusterapi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

var t = framework.NewTestFramework("capi_overrides")

var _ = t.Describe("Cluster API Overrides", Label("f:platform-lcm.install"), func() {
	var clientset dynamic.Interface

	// Get dynamic client
	Eventually(func() (dynamic.Interface, error) {
		kubePath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return nil, err
		}
		clientset, err = pkg.GetDynamicClientInCluster(kubePath)
		return clientset, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	// Get the components from the BOM
	rancherComp, capiComp := getComponents()
	Expect(capiComp.Version).To(Equal("v1.6.1"))
	Expect(rancherComp.Version).To(Equal("v2.7.3"))

	t.Context("initial state", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check the kontainerdrivers
		// THEN we successfully find them all active
		capipkg.WhenClusterAPIInstalledIt(t, "kontainerdrivers are active", func() {
			Eventually(capipkg.IsAllDriversActive(t, clientset), waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override ocneBootstrap and ocneControlPlane version", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override ocneBootstrap and ocneControlPlane versions
		// THEN the overrides get successfully applied
		capipkg.WhenClusterAPIInstalledIt(t, "kontainerdrivers are active", func() {
			Eventually(capipkg.IsAllDriversActive(t, clientset), waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

})

func getComponents() (*bom.BomComponent, *bom.BomComponent) {
	// Get the BOM from the installed Platform Operator
	bomDoc, err := pkg.GetBOMDoc()
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed to get BOM from platform operator: %v", err))
	}

	// Find the Rancher and CAPI components
	var rancherComp *bom.BomComponent
	var capiComp *bom.BomComponent
	for i, component := range bomDoc.Components {
		switch component.Name {
		case "rancher":
			rancherComp = &bomDoc.Components[i]
		case "capi-ocne":
			capiComp = &bomDoc.Components[i]
		}
	}
	Expect(rancherComp).To(Not(BeNil()))
	Expect(capiComp).To(Not(BeNil()))
	return rancherComp, capiComp
}
