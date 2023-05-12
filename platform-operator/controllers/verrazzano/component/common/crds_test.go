// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsApplyCRDYamlValid tests the applyCRDYaml function
// GIVEN a call to ApplyCRDYaml
// WHEN the yaml is valid
// THEN no error is returned
func TestIsApplyCRDYamlValid(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	config.TestHelmConfigDir = "../../../../helm_config"
	assert.Nil(t, ApplyCRDYaml(spi.NewFakeContext(fakeClient, nil, nil, false), config.GetHelmAppOpChartsDir()))
}

// TestIsApplyCRDYamlInvalidPath tests the applyCRDYaml function
// GIVEN a call to ApplyCRDYaml
// WHEN the path is invalid
// THEN an appropriate error is returned
func TestIsApplyCRDYamlInvalidPath(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	config.TestHelmConfigDir = "./testdata"
	assert.Error(t, ApplyCRDYaml(spi.NewFakeContext(fakeClient, nil, nil, false), ""))
}

// TestIsApplyCRDYamlInvalidChart tests the applyCRDYaml function
// GIVEN a call to ApplyCRDYaml
// WHEN the yaml is invalid
// THEN an appropriate error is returned
func TestIsApplyCRDYamlInvalidChart(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	config.TestHelmConfigDir = "invalidPath"
	assert.Error(t, ApplyCRDYaml(spi.NewFakeContext(fakeClient, nil, nil, false), ""))
}

// TestConvertVerrazzanoCR tests the ConvertVerrazzanoCR function
func TestConvertVerrazzanoCR(t *testing.T) {
	vz := vzapi.Verrazzano{}
	convertVZ := v1beta1.Verrazzano{}
	config.TestHelmConfigDir = "invalidPath"
	// GIVEN old Vz and a newer version of Vz
	// WHEN the ConvertVerrazzanoCR is called
	// THEN no error is returned
	err := ConvertVerrazzanoCR(&vz, &convertVZ)
	assert.Nil(t, err)

	// GIVEN a nil old Vz and a valid new Vz
	// WHEN the ConvertVerrazzanoCR is called
	// THEN an appropriate error is returned
	err = ConvertVerrazzanoCR(nil, &convertVZ)
	assert.Error(t, err)
}

// TestCheckCRDsExist tests the CheckCRDsExist function
// GIVEN a call to CheckCRDsExist
// WHEN the requested CRDs are or aren't present
// THEN true is returned if they are, false if not
func TestCheckCRDsExist(t *testing.T) {
	asserts := assert.New(t)

	testCRDs := []string{
		"foo",
		"bar",
	}
	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return nil, fmt.Errorf("unexpected error")
	}

	exist, err := CheckCRDsExist(testCRDs)
	asserts.Error(err)
	asserts.False(exist)

	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset().ApiextensionsV1(), nil
	}

	exist, err = CheckCRDsExist(testCRDs)
	asserts.NoError(err)
	asserts.False(exist)

	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(newTestCRDs(testCRDs...)...).ApiextensionsV1(), nil
	}

	exist, err = CheckCRDsExist(testCRDs)
	asserts.NoError(err)
	asserts.True(exist)
}

func newTestCRDs(crds ...string) []runtime.Object {
	var runtimeObjs []runtime.Object
	for _, crd := range crds {
		runtimeObjs = append(runtimeObjs, newCRD(crd))
	}
	return runtimeObjs
}

func newCRD(name string) runtime.Object {
	crd := &v1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return crd
}
