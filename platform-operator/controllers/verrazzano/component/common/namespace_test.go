// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestCreateAndLabelNamespaces tests the CreateAndLabelNamespaces function
// GIVEN a component context
// WHEN  the CreateAndLabelNamespaces function is called
// THEN  the function call succeeds and the expected namespace has been created and labelled
func TestCreateAndLabelNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	a := true
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Velero: &vzapi.VeleroComponent{
					Enabled: &a,
				},
			},
		},
	}

	ctx := spi.NewFakeContext(client, vz, nil, false)
	err := CreateAndLabelNamespaces(ctx)
	assert.NoError(t, err)

}

// TestCheckExistingNamespace tests the CheckExistingNamespace function
func TestCheckExistingNamespace(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	var list = []corev1.Namespace{}
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "namespace1"}}
	list = append(list, ns)

	// GIVEN a list of namespaces and a namespace that doesn't exist
	// WHEN  the CheckExistingNamespace function is called
	// THEN  the function call fails when namespace doesn't exist
	err := CheckExistingNamespace(list, isRancherNamespace)
	assert.Error(t, err)

	// GIVEN a component context and a Verrazzano CR
	// WHEN  the CheckExistingNamespace function is called
	// THEN  the function call succeeds with no error
	err = CheckExistingNamespace(nil, isRancherNamespace)
	assert.NoError(t, err)

}

// isRancherNamespace determines whether the namespace given is a Rancher ns
func isRancherNamespace(ns *corev1.Namespace) bool {
	var rancherSystemNS = []string{
		"namespace1",
		"cattle-alerting"}
	if vzstring.SliceContainsString(rancherSystemNS, ns.Name) {
		return true
	}
	if ns.Annotations == nil {
		return false
	}
	if val, ok := ns.Annotations["rancherSysNS"]; ok && val == "true" {
		return true
	}
	return false
}
