// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetLatestReleaseVersion
// GIVEN a list of release versions
//  WHEN I call this function
//  THEN expect it to return the latest version string
func TestGetLatestReleaseVersion(t *testing.T) {

	releases := []string{"v0.1.0", "v1.2.1", "v1.3.1"}
	latestRelease, err := getLatestReleaseVersion(releases)
	assert.NoError(t, err)
	assert.Equal(t, latestRelease, "v1.3.1")
}

// TestGetVerrazzanoResource
// GIVEN the namespace and name of a verrazzano resource
//  WHEN I call this function
//  THEN expect it to return a verrazzano rsource
func TestGetVerrazzanoResource(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
		}).Build()

	vz, err := GetVerrazzanoResource(client, types.NamespacedName{Namespace: "default", Name: "verrazzano"})
	assert.NoError(t, err)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, "verrazzano", vz.Name)
}

// TestGetVerrazzanoResourceNotFound
// GIVEN the namespace and name of a verrazzano resource
//  WHEN I call this function
//  THEN expect it to return an error
func TestGetVerrazzanoResourceNotFound(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	_, err := GetVerrazzanoResource(client, types.NamespacedName{Namespace: "default", Name: "verrazzano"})
	assert.EqualError(t, err, "verrazzanos.install.verrazzano.io \"verrazzano\" not found")
}
