// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetLatestReleaseVersion
// GIVEN a list of release versions
//  WHEN I call this function
//  THEN expect it to return the latest version string
func TestGetLatestReleaseVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	latestRelease, err := GetLatestReleaseVersion(rc.GetHTTPClient())
	assert.NoError(t, err)
	assert.Equal(t, latestRelease, "v1.3.1")
}

// TestGetVerrazzanoResource
// GIVEN the namespace and name of a verrazzano resource
//  WHEN I call GetVerrazzanoResource
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
//  WHEN I call GetVerrazzanoResource
//  THEN expect it to return an error
func TestGetVerrazzanoResourceNotFound(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	_, err := GetVerrazzanoResource(client, types.NamespacedName{Namespace: "default", Name: "verrazzano"})
	assert.EqualError(t, err, "verrazzanos.install.verrazzano.io \"verrazzano\" not found")
}

// TestFindVerrazzanoResource
// GIVEN a list of a verrazzano resources
//  WHEN I call FindVerrazzanoResource
//  THEN expect to find a single verrazzano rsource
func TestFindVerrazzanoResource(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
		}).Build()

	vz, err := FindVerrazzanoResource(client)
	assert.NoError(t, err)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, "verrazzano", vz.Name)
}

// TestFindVerrazzanoResourceMultiple
// GIVEN a list of a verrazzano resources
//  WHEN I call FindVerrazzanoResource
//  THEN return an error when multiple verrazzano resources found
func TestFindVerrazzanoResourceMultiple(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
		},
		&vzapi.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano2",
			},
		}).Build()

	_, err := FindVerrazzanoResource(client)
	assert.EqualError(t, err, "Expected to only find one Verrazzano resource, but found 2")
}

// TestFindVerrazzanoResourceNone
// GIVEN a list of a verrazzano resources
//  WHEN I call FindVerrazzanoResource
//  THEN return an error when no verrazzano resources are found
func TestFindVerrazzanoResourceNone(t *testing.T) {
	_ = vzapi.AddToScheme(k8scheme.Scheme)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	_, err := FindVerrazzanoResource(client)
	assert.EqualError(t, err, "Failed to find any Verrazzano resources")
}
