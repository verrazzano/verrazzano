// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
	v1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestNewVerrazzanoForVersion
// GIVEN a schema.GroupVersion
//
//	WHEN I call this function
//	THEN expect it to return a function that returns a new Verrazzano of the appropriate version
func TestNewVerrazzanoForVersion(t *testing.T) {
	var tests = []struct {
		name string
		gv   schema.GroupVersion
		o    interface{}
	}{
		{
			"new v1alpha1 Verrazzano",
			v1alpha1.SchemeGroupVersion,
			&v1alpha1.Verrazzano{},
		},
		{
			"new v1beta1 Verrazzano",
			v1beta1.SchemeGroupVersion,
			&v1beta1.Verrazzano{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.o, NewVerrazzanoForGroupVersion(tt.gv)())
		})
	}
}

// TestGetLatestReleaseVersion
// GIVEN a list of release versions
//
//	WHEN I call this function
//	THEN expect it to return the latest version string
func TestGetLatestReleaseVersion(t *testing.T) {
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	latestRelease, err := GetLatestReleaseVersion(rc.GetHTTPClient())
	assert.NoError(t, err)
	assert.Equal(t, latestRelease, "v1.3.1")
}

// TestGetVerrazzanoResource
// GIVEN the namespace and name of a verrazzano resource
//
//	WHEN I call GetVerrazzanoResource
//	THEN expect it to return a verrazzano resource
func TestVerrazzanoResource(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(NewScheme()).WithObjects(
		&v1beta1.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
			Status: v1beta1.VerrazzanoStatus{
				Components: map[string]*v1beta1.ComponentStatusDetails{"test_component": {Name: "grafana"}},
			},
		}).Build()

	vz, err := GetVerrazzanoResource(client, types.NamespacedName{Namespace: "default", Name: "verrazzano"})
	assert.NoError(t, err)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, "verrazzano", vz.Name)
	assert.Nil(t, UpdateVerrazzanoResource(client, vz))
	assert.NotEmpty(t, GetNamespacesForAllComponents(*vz))
	_, err = findVerazzanoResourceV1Alpha1(client)
	assert.Error(t, failedToFindResourceError(err))
}

// TestGetVerrazzanoResourceNotFound
// GIVEN the namespace and name of a verrazzano resource
//
//	WHEN I call GetVerrazzanoResource
//	THEN expect it to return an error
func TestGetVerrazzanoResourceNotFound(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(NewScheme()).Build()
	_, err := GetVerrazzanoResource(client, types.NamespacedName{Namespace: "default", Name: "verrazzano"})
	assert.EqualError(t, err, "Failed to get a Verrazzano install resource: verrazzanos.install.verrazzano.io \"verrazzano\" not found")
}

// TestFindVerrazzanoResource
// GIVEN a list of a verrazzano resources
//
//	WHEN I call FindVerrazzanoResource
//	THEN expect to find a single verrazzano resource
func TestFindVerrazzanoResource(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(NewScheme()).WithObjects(
		&v1beta1.Verrazzano{
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
//
//	WHEN I call FindVerrazzanoResource
//	THEN return an error when multiple verrazzano resources found
func TestFindVerrazzanoResourceMultiple(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(NewScheme()).WithObjects(
		&v1beta1.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
		},
		&v1beta1.Verrazzano{
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
//
//	WHEN I call FindVerrazzanoResource
//	THEN return an error when no verrazzano resources are found
func TestFindVerrazzanoResourceNone(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(NewScheme()).Build()

	_, err := FindVerrazzanoResource(client)
	assert.EqualError(t, err, "Failed to find any Verrazzano resources")
}

// TestGetNamespacesForAllComponents
// GIVEN a Verrazzano resource
//
//	WHEN I call GetNamespacesForAllComponents
//	THEN return list of namespaces of all components in that Verrazzano resource
func TestGetNamespacesForAllComponents(t *testing.T) {
	//components:
	vzWithOneComponent :=
		v1beta1.Verrazzano{
			Status: v1beta1.VerrazzanoStatus{
				Components: v1beta1.ComponentStatusMap{
					constants.Grafana: &v1beta1.ComponentStatusDetails{
						Name:  constants.Grafana,
						State: v1beta1.CompStateReady,
					},
				},
			},
		}

	vzWithMultipleComponents :=
		v1beta1.Verrazzano{
			Status: v1beta1.VerrazzanoStatus{
				Components: v1beta1.ComponentStatusMap{
					constants.ExternalDNS: &v1beta1.ComponentStatusDetails{
						Name:  constants.ExternalDNS,
						State: v1beta1.CompStateReady,
					},
					constants.Grafana: &v1beta1.ComponentStatusDetails{
						Name:  constants.Grafana,
						State: v1beta1.CompStateReady,
					},
					constants.Istio: &v1beta1.ComponentStatusDetails{
						Name:  constants.Istio,
						State: v1beta1.CompStateReady,
					},
				},
			},
		}

	vzWithDisabledComponent :=
		v1beta1.Verrazzano{
			Status: v1beta1.VerrazzanoStatus{
				Components: v1beta1.ComponentStatusMap{
					constants.ExternalDNS: &v1beta1.ComponentStatusDetails{
						Name:  constants.ExternalDNS,
						State: v1beta1.CompStateReady,
					},
					constants.Grafana: &v1beta1.ComponentStatusDetails{
						Name:  constants.Grafana,
						State: v1beta1.CompStateReady,
					},
					constants.Istio: &v1beta1.ComponentStatusDetails{
						Name:  constants.Istio,
						State: v1beta1.CompStateDisabled,
					},
				},
			},
		}

	var tests = []struct {
		name      string
		component v1beta1.Verrazzano
		expected  []string
	}{
		{
			"vz with one component",
			vzWithOneComponent,
			[]string{constants.VerrazzanoSystemNamespace},
		},
		{
			"vz with multiple components",
			vzWithMultipleComponents,
			[]string{constants.IstioSystemNamespace, constants.VerrazzanoSystemNamespace, constants.CertManagerNamespace},
		},
		{
			"vz with disabled component",
			vzWithDisabledComponent,
			[]string{constants.VerrazzanoSystemNamespace, constants.CertManagerNamespace},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetNamespacesForAllComponents(tt.component)
			for i := range tt.expected {
				assert.Contains(t, result, tt.expected[i])
			}
			assert.Equal(t, len(tt.expected), len(result))
		})
	}
}

// TestNewVerrazzanoForVZVersion
// GIVEN a version
//
//	WHEN I call NewVerrazzanoForVersion
//	THEN expect it to return a new Verrazzano of the appropriate version
func TestNewVerrazzanoForVZVersion(t *testing.T) {
	var tests = []struct {
		name    string
		version string
		gv      schema.GroupVersion
	}{
		{
			"new v1alpha1 Verrazzano with default version",
			"1.3.0",
			v1alpha1.SchemeGroupVersion,
		},
		{
			"new v1beta1 Verrazzano with min version",
			"1.4.0",
			v1beta1.SchemeGroupVersion,
		},
		{
			"new v1beta1 Verrazzano below min version",
			"1.2.0",
			v1alpha1.SchemeGroupVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gv, obj, _ := NewVerrazzanoForVZVersion(tt.version)
			version := obj.GetObjectKind().GroupVersionKind().Version
			assert.Equal(t, tt.gv.Version, gv.Version)
			assert.Equal(t, tt.gv.Group, gv.Group)
			assert.Equal(t, version, gv.Version)
		})
	}
}

// TestUpdateVerrazzanoResource
// GIVEN a Verrazzano resource and specified group version
//
//	WHEN I call UpdateVerrazzanoResource
//	THEN expect it to update that version of the Verrazzano resource, an error will be returned if it's not supported
func TestUpdateVerrazzanoResource(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(NewScheme()).WithObjects(
		&v1beta1.Verrazzano{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
		}).Build()
	vz, _ := GetVerrazzanoResource(client, types.NamespacedName{Namespace: "default", Name: "verrazzano"})
	err := UpdateVerrazzanoResource(client, vz)
	assert.NoError(t, err)

}

// TestGetOperatorYaml
// GIVEN a specified cersion
//
//	WHEN I call GetOperatorYaml
//	THEN expect it to return Kubernetes manifests to deploy the Verrazzano platform operator
func TestGetOperatorYaml(t *testing.T) {
	var tests = []struct {
		name     string
		version  string
		expected string
	}{
		{
			"earlier than 1.4.0",
			"1.3.0",
			"https://github.com/verrazzano/verrazzano/releases/download/1.3.0/operator.yaml",
		},
		{
			"later than 1.4.0",
			"1.5.0",
			"https://github.com/verrazzano/verrazzano/releases/download/1.5.0/verrazzano-platform-operator.yaml",
		},
		{
			"equal to 1.4.0",
			"1.4.0",
			"https://github.com/verrazzano/verrazzano/releases/download/1.4.0/verrazzano-platform-operator.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, _ := GetOperatorYaml(tt.version)
			assert.Equal(t, str, tt.expected)
		})
	}
}
