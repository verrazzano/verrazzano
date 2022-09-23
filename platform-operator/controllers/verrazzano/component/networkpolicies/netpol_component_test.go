// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// GIVEN a network policies helm component
//  WHEN the IsEnabled function is called
//  THEN the call always returns true
func TestIsEnabled(t *testing.T) {
	comp := NewComponent()
	assert.True(t, comp.IsEnabled(nil))
}

// GIVEN a network policies helm component
//  WHEN the PreInstall function is called
//  THEN the expected namespaces have been created
func TestPreInstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	comp := NewComponent()

	err := comp.PreInstall(ctx)
	assert.NoError(t, err)
	assertNamespaces(t, fakeClient)
}

// GIVEN a network policies helm component
//  WHEN the PostInstall function is called
//  THEN the call returns no error
func TestPostInstall(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	comp := NewComponent()

	err := comp.PostInstall(ctx)
	assert.NoError(t, err)
}

// GIVEN a network policies helm component
//  WHEN the PreUpgrade function is called
//   AND there is an existing network policy associated with the verrazzano helm release
//  THEN the network policy association is changed so that it is associated with the network policies helm release
func TestPreUpgrade(t *testing.T) {
	const netPolName = "istiod-access"
	fakeClient := fake.NewClientBuilder().WithObjects(
		&netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: constants.IstioSystemNamespace, Name: netPolName}},
	).Build()

	// associate the network policy with the verrazzano helm release
	obj := &netv1.NetworkPolicy{}
	netPolNSN := types.NamespacedName{Namespace: constants.IstioSystemNamespace, Name: netPolName}
	// importing verrazzano component package results in import cycle so doing it like this instead
	vzComponentNSN := types.NamespacedName{Namespace: constants.VerrazzanoSystemNamespace, Name: "verrazzano"}

	_, err := common.AssociateHelmObject(fakeClient, obj, vzComponentNSN, netPolNSN, false)
	assert.NoError(t, err)

	ctx := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	comp := NewComponent()

	err = comp.PreUpgrade(ctx)
	assert.NoError(t, err)
	assertNamespaces(t, fakeClient)

	// assert that the network policy is now associated with this component's helm release
	assertNetPolicyHelmOwnership(t, fakeClient)
}

// GIVEN a network policies helm component
//  WHEN the PostUpgrade function is called
//  THEN the call returns no error
func TestPostUpgrade(t *testing.T) {
	fakeClient := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	comp := NewComponent()

	err := comp.PostUpgrade(ctx)
	assert.NoError(t, err)
}

// GIVEN a network policies helm component
//  WHEN the PreUninstall function is called
//  THEN the expected namespaces have been created
func TestPreUninstall(t *testing.T) {
	const netPolName = "istiod-access"
	fakeClient := fake.NewClientBuilder().WithObjects(
		&netv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.IstioSystemNamespace,
				Name:      netPolName,
				Annotations: map[string]string{
					"helm.sh/resource-policy": "keep",
				},
			},
		},
	).Build()
	ctx := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)
	comp := NewComponent()

	err := comp.PreUninstall(ctx)
	assert.NoError(t, err)

	// validate that the annotation has been removed
	netpol := &netv1.NetworkPolicy{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Namespace: constants.IstioSystemNamespace, Name: netPolName}, netpol)
	assert.NoError(t, err)
	assert.Empty(t, netpol.Annotations)
}

// assertNamespaces asserts that all namespaces with Verrazzano network policies exist
func assertNamespaces(t *testing.T, client clipkg.Client) {
	nsObj := &corev1.Namespace{}
	for _, ns := range netPolNamespaces() {
		err := client.Get(context.TODO(), types.NamespacedName{Name: ns}, nsObj)
		assert.NoErrorf(t, err, "Expected namespace %s to exist", ns)
	}
}

// assertNetPolicyHelmOwnership asserts that any network policies are now associated with the helm release
// in this component
func assertNetPolicyHelmOwnership(t *testing.T, client clipkg.Client) {
	found := false
	for _, ns := range netPolNamespaces() {
		netpolList := netv1.NetworkPolicyList{}
		client.List(context.TODO(), &netpolList, &clipkg.ListOptions{Namespace: ns})

		for _, netpol := range netpolList.Items {
			found = true
			assert.Equal(t, ComponentName, netpol.Annotations["meta.helm.sh/release-name"])
			assert.Equal(t, ComponentNamespace, netpol.Annotations["meta.helm.sh/release-namespace"])
			assert.Equal(t, "Helm", netpol.Labels["app.kubernetes.io/managed-by"])
		}
	}

	assert.True(t, found, "Expected to find at least one network policy")
}

// netPolNamespaces returns an array of namespace names that contain Verrazzano network policies
func netPolNamespaces() []string {
	// collect unique namespace names from netpolNamespaceNames
	nsnMap := make(map[string]bool)
	for _, nsn := range netpolNamespaceNames {
		nsnMap[nsn.Namespace] = true
	}

	// convert to array of namespace names
	var namespaces []string
	for key := range nsnMap {
		namespaces = append(namespaces, key)
	}

	return namespaces
}
