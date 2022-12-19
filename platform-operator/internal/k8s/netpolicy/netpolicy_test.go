// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package netpolicy

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"testing"
)

// TestCreateNetworkPolicies tests creating network policies for the operator.
// GIVEN a call to CreateOrUpdateNetworkPolicies
// WHEN the network policies do not exist
// THEN the network policies are created
func TestCreateNetworkPolicies(t *testing.T) {
	asserts := assert.New(t)
	mockClient := ctrlfake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// create the network policy
	opResult, errors := CreateOrUpdateNetworkPolicies(mockClient)
	asserts.Empty(errors)
	asserts.Contains(opResult, controllerutil.OperationResultCreated)

	// fetch the policy and make sure the spec matches what we expect
	netPolicy := &netv1.NetworkPolicy{}
	err := mockClient.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoInstallNamespace, Name: networkPolicyPodName}, netPolicy)
	asserts.NoError(err)

	expectedNetPolicies := newNetworkPolicies()
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
	port := intstr.FromInt(9100)
	netpol := &netv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: networkPolicyAPIVersion,
			Kind:       networkPolicyKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoInstallNamespace,
			Name:      networkPolicyPodName,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					podAppLabel: networkPolicyPodName,
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
				netv1.PolicyTypeEgress,
			},
			Egress: []netv1.NetworkPolicyEgressRule{
				{
					Ports: []netv1.NetworkPolicyPort{
						{
							Port: &port,
						},
					},
				},
			},
			Ingress: []netv1.NetworkPolicyIngressRule{
				{
					From: []netv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									verrazzanoNamespaceLabel: constants.VerrazzanoMonitoringNamespace,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									appNameLabel: constants.PrometheusStorageLabelValue,
								},
							},
						},
					},
				},
			},
		},
	}

	mockClient := ctrlfake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(netpol).Build()

	// this call should update the network policy
	opResult, errors := CreateOrUpdateNetworkPolicies(mockClient)
	asserts.Empty(errors)
	asserts.Contains(opResult, controllerutil.OperationResultUpdated)

	// fetch the policy and make sure the spec matches what we expect
	netPolicy := &netv1.NetworkPolicy{}
	err := mockClient.Get(context.TODO(), client.ObjectKey{Namespace: constants.VerrazzanoInstallNamespace, Name: networkPolicyPodName}, netPolicy)
	asserts.NoError(err)

	expectedNetPolicies := newNetworkPolicies()
	var expectedSpecs []netv1.NetworkPolicySpec
	for _, netpol := range expectedNetPolicies {
		expectedSpecs = append(expectedSpecs, netpol.Spec)
	}
	asserts.Contains(expectedSpecs, netPolicy.Spec)
}
