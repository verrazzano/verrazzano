// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	apiyaml "k8s.io/apimachinery/pkg/util/yaml"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var (
	testRef = vmcv1alpha1.NamespacedRef{
		Name:      "test",
		Namespace: "test",
	}
	scheme *runtime.Scheme
	//go:embed testdata/identity-base.yaml
	testIdentityBase []byte
	//go:embed testdata/patch-allow-all.yaml
	testPatchAllowAll []byte
	//go:embed testdata/patch-allow-default-ns.yaml
	testPatchAllowDefaultNs []byte
	//go:embed testdata/patch-allow-test-ns.yaml
	testPatchAllowTestNs []byte
	//go:embed testdata/patch-allow-test-ns-by-selector.yaml
	testPatchAllowTestNsBySelector []byte
)

func init() {
	scheme = runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
}

func TestLoadCredentials(t *testing.T) {
	selectorNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testRef.Name,
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}
	clusterNamespace := testRef.Namespace
	var tests = []struct {
		name     string
		cli      clipkg.Client
		hasError bool
	}{
		{
			"access when identity allows test namespace by selector",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(newTestIdentity(testPatchAllowTestNsBySelector), newTestSecret(), selectorNamespace).Build(),
			false,
		},
		{
			"access when identity allows test namespace",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(newTestIdentity(testPatchAllowTestNs), newTestSecret()).Build(),
			false,
		},
		{
			"deny when identity allows a namespace that isn't the test namespaces",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(newTestIdentity(testPatchAllowDefaultNs), newTestSecret()).Build(),
			true,
		},
		{
			"allow when identity allows all namespaces",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(newTestIdentity(testPatchAllowAll), newTestSecret()).Build(),
			false,
		},
		{
			"deny when identity has no allowedNamespaces",
			fake.NewClientBuilder().WithScheme(scheme).WithObjects(newTestIdentity(nil)).Build(),
			true,
		},
		{
			"deny when identity not found",
			fake.NewClientBuilder().WithScheme(scheme).Build(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := oci.CredentialsLoaderImpl{}.GetCredentialsIfAllowed(context.TODO(), tt.cli, testRef.AsNamespacedName(), clusterNamespace)
			if tt.hasError {
				assert.Error(t, err)
				assert.Nil(t, c)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c)
			}
		})
	}
}

func newTestIdentity(allowedNamespaces []byte) *unstructured.Unstructured {
	j, _ := apiyaml.ToJSON(testIdentityBase)
	obj, _ := runtime.Decode(unstructured.UnstructuredJSONScheme, j)
	identity := obj.(*unstructured.Unstructured)
	if len(allowedNamespaces) > 0 {
		allowedNamespacesJSON, _ := apiyaml.ToJSON(allowedNamespaces)
		m := map[string]interface{}{}
		_ = json.Unmarshal(allowedNamespacesJSON, &m)
		identity.Object["spec"].(map[string]interface{})["allowedNamespaces"] = m
	}
	return identity
}

func newTestSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testRef.Name,
			Namespace: testRef.Namespace,
		},
		Data: map[string][]byte{},
	}
}
