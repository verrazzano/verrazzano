// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var testProject = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
		Template: ProjectTemplate{
			Namespaces: []NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "newNS1",
					},
				},
			},
		},
	},
}

var testNetworkPolicy = VerrazzanoProject{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "test",
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	},
	Spec: VerrazzanoProjectSpec{
		Template: ProjectTemplate{
			Namespaces: []NamespaceTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
			},
			NetworkPolicies: []NetworkPolicyTemplate{
				{
					Metadata: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "net1",
					},
					Spec: netv1.NetworkPolicySpec{},
				}},
		},
	},
}

// TestVerrazzanoProject tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject is properly formed
// THEN the validation should succeed
func TestVerrazzanoProject(t *testing.T) {
	// Test data
	testVP := testProject

	// Test create
	err := testVP.ValidateCreate()
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
}

// TestInvalidNamespace tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject contains an invalid namespace
// THEN the validation should fail
func TestInvalidNamespace(t *testing.T) {

	// Test data
	testVP := testProject
	testVP.Namespace = "invalid-namespace"

	// Test create
	err := testVP.ValidateCreate()
	assert.Error(t, err, "Expected failure for invalid namespace")
	assert.Containsf(t, err.Error(), fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.Error(t, err, "Expected failure for invalid namespace")
	assert.Containsf(t, err.Error(), fmt.Sprintf("resource must be %q", constants.VerrazzanoMultiClusterNamespace), "unexpected failure string")
}

// TestInvalidNamespaces tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject contains an invalid namespace list
// THEN the validation should fail
func TestInvalidNamespaces(t *testing.T) {

	// Test data
	testVP := testProject
	testVP.Spec.Template.Namespaces = []NamespaceTemplate{}

	// Test create
	err := testVP.ValidateCreate()
	assert.Error(t, err, "Expected failure for invalid namespace list")
	assert.Containsf(t, err.Error(), "One or more namespaces must be provided", "unexpected failure string")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.Error(t, err, "Expected failure for invalid namespace list")
	assert.Containsf(t, err.Error(), "One or more namespaces must be provided", "unexpected failure string")
}

// TestNetworkPolicyNamespace tests the validation of VerrazzanoProject NetworkPolicyTemplate
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a NetworkPolicyTemplate with a namespace that exists in the project
// THEN the validation should succeed
func TestNetworkPolicyNamespace(t *testing.T) {
	// Test data
	testVP := testNetworkPolicy

	// Test create
	err := testVP.ValidateCreate()
	assert.NoError(t, err, "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.NoError(t, err, "Error validating VerrazzanProject with NetworkPolicyTemplate")
}

// TestNetworkPolicyMissingNamespace tests the validation of VerrazzanoProject NetworkPolicyTemplate
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a NetworkPolicyTemplate with a namespace that does not exist in the project
// THEN the validation should fail
func TestNetworkPolicyMissingNamespace(t *testing.T) {
	// Test data
	testVP := testNetworkPolicy
	testVP.Spec.Template.Namespaces[0].Metadata.Name = "ns2"

	// Test create
	err := testVP.ValidateCreate()
	assert.EqualError(t, err, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.EqualError(t, err, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")
}
