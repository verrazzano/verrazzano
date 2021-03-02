// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
