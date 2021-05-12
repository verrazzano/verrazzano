// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

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

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

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

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

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

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

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

	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

	// Test create
	err := testVP.ValidateCreate()
	assert.EqualError(t, err, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")

	// Test update
	err = testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.EqualError(t, err, "namespace ns1 used in NetworkPolicy net1 does not exist in project", "Error validating VerrazzanProject with NetworkPolicyTemplate")
}

// TestNamespaceUniquenessForProjects tests that the namespace of a VerrazzanoProject N does not conflict with a preexisting project
// GIVEN a call validate VerrazzanoProject on create or update
// WHEN the VerrazzanoProject has a a namespace that conflicts with any pre-existing projects
// THEN the validation should fail
func TestNamespaceUniquenessForProjects(t *testing.T) {

	// When creating the fake client, prepopulate it with 2 Verrazzano projects
	// existingVP1 has namespaces project1 and project2
	// existingVP2 has namespaces project3 and project4
	// Adding any new Verrazzano projects with these namespaces will fail validation
	existingVP1 := &VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project1",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project2",
						},
					},
				},
			},
		},
	}

	existingVP2 := &VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-project-2",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project3",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project4",
						},
					},
				},
			},
		},
	}

	objs := []runtime.Object{existingVP1, existingVP2}
	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme(), objs...), nil
	}
	defer func() { getControllerRuntimeClient = getClient }()

	currentVP := VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project",
						},
					},
				},
			},
		},
	}
	// This test will succeed because Verrazzano project test-project has unique namespace project
	err := currentVP.validateNamespaceCanBeUsed()
	assert.Nil(t, err)

	currentVP = VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project1",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project2",
						},
					},
				},
			},
		},
	}

	// This test will fail because Verrazzano project test-project1 has conflicting namespace project2
	err = currentVP.validateNamespaceCanBeUsed()
	assert.NotNil(t, err)

	currentVP = VerrazzanoProject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project2",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoProjectSpec{
			Template: ProjectTemplate{
				Namespaces: []NamespaceTemplate{
					{
						Metadata: metav1.ObjectMeta{
							Name: "project",
						},
					},
					{
						Metadata: metav1.ObjectMeta{
							Name: "project4",
						},
					},
				},
			},
		},
	}
	// This test will fail because Verrazzano project test-project2 has conflicting namespace project4
	err = currentVP.validateNamespaceCanBeUsed()
	assert.NotNil(t, err)
}
