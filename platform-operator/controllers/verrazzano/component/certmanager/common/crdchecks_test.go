// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1fake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

// TestGetRequiredCertManagerCRDNames tests the GetRequiredCertManagerCRDNames function
// GIVEN a call to GetRequiredCertManagerCRDNames
// THEN the correct number of strings are returned
func TestGetRequiredCertManagerCRDNames(t *testing.T) {
	crdNames := GetRequiredCertManagerCRDNames()
	assert.Len(t, crdNames, 5)
}

// TestCertManagerCrdsExist tests the CertManagerCRDsExist function
// GIVEN a call to CertManagerCRDsExist
// THEN false is returned if the CRDs do not exist, true otherwise
func TestCertManagerCrdsExist(t *testing.T) {
	asserts := assert.New(t)

	defer func() { k8sutil.ResetGetAPIExtV1ClientFunc() }()
	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return nil, fmt.Errorf("unexpected error")
	}

	crdsExist, err := CertManagerCRDsExist()
	asserts.False(crdsExist)
	asserts.Error(err)

	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset().ApiextensionsV1(), nil
	}

	crdsExist, err = CertManagerCRDsExist()
	asserts.False(crdsExist)
	asserts.NoError(err)

	k8sutil.GetAPIExtV1ClientFunc = func() (apiextv1.ApiextensionsV1Interface, error) {
		return apiextv1fake.NewSimpleClientset(createCertManagerCRDs()...).ApiextensionsV1(), nil
	}

	crdsExist, err = CertManagerCRDsExist()
	asserts.True(crdsExist)
	asserts.NoError(err)
}

func createCertManagerCRDs() []runtime.Object {
	var runtimeObjs []runtime.Object
	for _, crd := range GetRequiredCertManagerCRDNames() {
		runtimeObjs = append(runtimeObjs, newCRD(crd))
	}
	return runtimeObjs
}

func newCRD(name string) clipkg.Object {
	crd := &v1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return crd
}
