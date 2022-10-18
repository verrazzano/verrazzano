// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpolicy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	apiServerIP   = "1.2.3.4"
	apiServerPort = 6443
)

// TestCreateNetworkPolicies tests creating network policies for the operator.
// GIVEN a call to CreateOrUpdateNetworkPolicies
// WHEN the network policies do not exist
// THEN the network policies are created
func TestCreateNetworkPolicies(t *testing.T) {
	asserts := assert.New(t)
	mockClient := ctrlfake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// mock the clientset with a kubernetes API server endpoint
	mockClientset := k8sfake.NewSimpleClientset(makeKubeAPIServerEndpoint())

	// create the network policy
	opResult, errors := CreateOrUpdateNetworkPolicies(mockClientset, mockClient)
	asserts.Empty(errors)
	asserts.Contains(opResult, controllerutil.OperationResultCreated)

	// fetch the policy and make sure the spec matches what we expect
	netPolicy := &netv1.NetworkPolicy{}
	err := mockClient.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoInstallNamespace, Name: networkPolicyPodName}, netPolicy)
	asserts.NoError(err)

	expectedNetPolicies := newNetworkPolicies(apiServerIP, apiServerPort)
	var expectedSpecs []netv1.NetworkPolicySpec
	for _, netpol := range expectedNetPolicies {
		expectedSpecs = append(expectedSpecs, netpol.Spec)
	}
	asserts.Contains(expectedSpecs, netPolicy.Spec)
}

// TestUpdateNetworkPolicies tests updating network policies for the operator.
// GIVEN a call to CreateOrUpdateNetworkPolicies
// WHEN the network policies already exist
// THEN the network policies are updated
func TestUpdateNetworkPolicies(t *testing.T) {
	asserts := assert.New(t)
	mockClient := ctrlfake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// mock the clientset with a kubernetes API server endpoint
	mockClientset := k8sfake.NewSimpleClientset(makeKubeAPIServerEndpoint())

	// make an existing network policy and change the API server IP
	existingNetPolicies := newNetworkPolicies("1.1.1.1", apiServerPort)
	for _, netpol := range existingNetPolicies {
		err := mockClient.Create(context.TODO(), netpol)
		if err != nil {
			return
		}
	}

	// this call should update the network policy
	opResult, errors := CreateOrUpdateNetworkPolicies(mockClientset, mockClient)
	asserts.Empty(errors)
	asserts.Contains(opResult, controllerutil.OperationResultUpdated)

	// fetch the policy and make sure the spec matches what we expect
	netPolicy := &netv1.NetworkPolicy{}
	err := mockClient.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoInstallNamespace, Name: networkPolicyPodName}, netPolicy)
	asserts.NoError(err)

	expectedNetPolicies := newNetworkPolicies(apiServerIP, apiServerPort)
	var expectedSpecs []netv1.NetworkPolicySpec
	for _, netpol := range expectedNetPolicies {
		expectedSpecs = append(expectedSpecs, netpol.Spec)
	}
	asserts.Contains(expectedSpecs, netPolicy.Spec)
}

// TestNetworkPoliciesFailures tests failure cases attempting to create or update
// the operator network policies.
func TestNetworkPoliciesFailures(t *testing.T) {
	asserts := assert.New(t)
	mockClient := ctrlfake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// GIVEN a call to CreateOrUpdateNetworkPolicies
	// WHEN there is no Kubernetes API server endpoint found
	// THEN we expect an error

	// mock the clientset, no kuberetes API server endpoints exist
	mockClientset := k8sfake.NewSimpleClientset()

	// this call should fail
	_, errors := CreateOrUpdateNetworkPolicies(mockClientset, mockClient)
	for _, err := range errors {
		asserts.Error(err)
	}

	// GIVEN a call to CreateOrUpdateNetworkPolicies
	// WHEN there is a Kubernetes API server endpoint
	// AND it does not contain any IP addresses or ports
	// THEN we expect an error

	emptyEndpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServerEndpointName,
			Namespace: corev1.NamespaceDefault,
		},
	}
	mockClientset = k8sfake.NewSimpleClientset(emptyEndpoints)

	// this call should fail
	_, errors = CreateOrUpdateNetworkPolicies(mockClientset, mockClient)
	for _, err := range errors {
		asserts.Error(err)
	}
}

// makeKubeAPIServerEndpoint returns a populated corev1.Endpoints struct with the kubernetes API server IP and port
func makeKubeAPIServerEndpoint() *corev1.Endpoints {
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServerEndpointName,
			Namespace: corev1.NamespaceDefault,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: apiServerIP,
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: apiServerPort,
					},
				},
			},
		},
	}
}
