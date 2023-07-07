// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi_overrides

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	capipkg "github.com/verrazzano/verrazzano/tests/e2e/pkg/clusterapi"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

const capiOverrides1 = `
{
  "defaultProviders": {
    "ocneBootstrap": {
      "version": "%s",
    },
    "ocneControlPlane": {
      "version": "%s",
    }
  }
}`

var t = framework.NewTestFramework("capi_overrides")

var _ = t.Describe("Cluster API Overrides", Label("f:platform-lcm.install"), func() {
	var dynClient dynamic.Interface

	// Get dynamic client
	Eventually(func() (dynamic.Interface, error) {
		kubePath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			return nil, err
		}
		dynClient, err = pkg.GetDynamicClientInCluster(kubePath)
		return dynClient, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

	// Get the components from the BOM
	_, capiComp := getComponents()

	t.Context("initial state", func() {
		// GIVEN the Cluster API is installed
		// WHEN we check the kontainerdrivers
		// THEN we successfully find them all active
		capipkg.WhenClusterAPIInstalledIt(t, "kontainerdrivers are active", func() {
			Eventually(capipkg.IsAllDriversActive(t, dynClient), waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("override ocneBootstrap and ocneControlPlane version", func() {
		// GIVEN the CAPI environment is ready
		// WHEN we override ocneBootstrap and ocneControlPlane versions
		// THEN the overrides get successfully applied

		capipkg.WhenClusterAPIInstalledIt(t, "kontainerdrivers are active", func() {
			applyOverrides(fmt.Sprintf(capiOverrides1, capiComp.Version, capiComp.Version))
			Eventually(capipkg.IsAllDriversActive(t, dynClient), waitTimeout, pollingInterval).Should(BeTrue())
		})
	})
})

// applyOverrides - apply overrides to CAPI component
func applyOverrides(overrides string) {
	// Get the VZ resource
	vz, err := pkg.GetVerrazzanoV1beta1()
	Expect(err).ToNot(HaveOccurred())

	// Get the client
	client, err := pkg.GetVerrazzanoClientset()
	Expect(err).ToNot(HaveOccurred())

	// Update the VZ with the overrides
	if vz.Spec.Components.ClusterAPI == nil {
		vz.Spec.Components.ClusterAPI = &v1beta1.ClusterAPIComponent{}
	}
	vz.Spec.Components.ClusterAPI.InstallOverrides = v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				Values: &apiextensionsv1.JSON{
					Raw: []byte(overrides),
				},
			},
		},
	}

	_, err = client.VerrazzanoV1beta1().Verrazzanos(vz.Namespace).Update(context.TODO(), vz, metav1.UpdateOptions{})
	Expect(err).ToNot(HaveOccurred())
}

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
