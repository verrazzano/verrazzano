// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
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
		Namespaces: []string{"bob"},
	},
}

// TestCreateVerrazzanoProject tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject create
// WHEN the VerrazzanoProject is properly formed
// THEN the validation should succeed
func TestCreateVerrazzanoProject(t *testing.T) {
	// Test data
	testVP := testProject
	err := testVP.ValidateCreate()
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
}

// TestUpdateVerrazzanoProject tests the validation of VerrazzanoProject resource
// GIVEN a call validate VerrazzanoProject update
// WHEN the VerrazzanoProject is properly formed
// THEN the validation should succeed
func TestUpdateVerrazzanoProject(t *testing.T) {

	// Test data
	testVP := testProject
	err := testVP.ValidateUpdate(&VerrazzanoProject{})
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
}
