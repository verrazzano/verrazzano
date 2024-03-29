// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakes "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestIsCloudCredentialSecret tests isOCNECloudCredential
// GIVEN Namespace resource
// WHEN isOCNECloudCredential is called
// THEN true is returned if secret is cloud credential in cattle global data namespace
// THEN false is returned if not cloud credential or is not in cattle global data namespace
func TestIsCloudCredentialSecret(t *testing.T) {
	asserts := assert.New(t)
	ccSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cloud-credential",
			Namespace: rancher.CattleGlobalDataNamespace,
		},
		Data: map[string][]byte{
			rancherCcFingerprintField: []byte("fingerprint"),
		},
	}
	nonCcSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cloud-credential",
			Namespace: rancher.CattleGlobalDataNamespace,
		},
	}
	ccInOtherNS := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cloud-credential",
			Namespace: "verrazzano-install",
		},
		Data: map[string][]byte{
			rancherCcFingerprintField: []byte("fingerprint"),
		},
	}
	asserts.True(isOCNECloudCredential(ccSecret))
	asserts.False(isOCNECloudCredential(nonCcSecret))
	asserts.False(isOCNECloudCredential(ccInOtherNS))
}

// TestUpdateOCNEclusterCloudCreds tests checkClusterCredentials
// GIVEN an updated cloud credential secret and the cluster's expired copy of that secret
// WHEN CheckClusterCredentials is called
// THEN the cluster's copy of the secret should be updated to match the new cloud credential
func TestCheckClusterCredentials(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, newClusterRepoResources()...)
	gvr := GetOCNEClusterAPIGVRForResource("clusters")
	// add dynamic elements to scheme
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: "Cluster" + "List"}, &unstructured.Unstructured{})
	ccSecretName := "test-secret"
	clusterSecretName := "cluster-principal" //nolint:gosec //#gosec G101
	ccSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ccSecretName,
			Namespace: rancher.CattleGlobalDataNamespace,
		},
		Data: map[string][]byte{
			rancherCcFingerprintField: []byte("fingerprint-new"),
			rancherCcTenancyField:     []byte("test-tenancy-new"),
			rancherCcRegionField:      []byte("test-region-new"),
		},
	}
	clusterSecretCopy := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterSecretName,
			Namespace: "cluster",
		},
		Data: map[string][]byte{
			ociCapiFingerprintField: []byte("fingerprint"),
			ociCapiTenancyField:     []byte("test-tenancy"),
			ociCapiRegionField:      []byte("test-region"),
		},
	}

	vz := &vzapi.Verrazzano{}

	fakeClient := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(ccSecret, vz, clusterSecretCopy).Build()
	r := &VerrazzanoSecretsReconciler{
		Client:        fakeClient,
		DynamicClient: dynamicClient,
		Scheme:        scheme,
		StatusUpdater: nil,
	}

	err := r.checkClusterCredentials(ccSecret)
	assert.NoError(t, err)
	updatedClusterSecretCopy := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: "cluster", Name: "cluster-principal"}, updatedClusterSecretCopy)
	assert.NoError(t, err)
	assert.Equalf(t, updatedClusterSecretCopy.Data[ociCapiFingerprintField], ccSecret.Data[rancherCcFingerprintField], "Expected fingerprint field of cloud credential copy to match updated cloud credential secret")
	assert.Equalf(t, updatedClusterSecretCopy.Data[ociCapiTenancyField], ccSecret.Data[rancherCcTenancyField], "Expected tenancy field of cloud credential copy to match updated cloud credential secret")
	assert.NotEqualf(t, updatedClusterSecretCopy.Data[ociCapiRegionField], ccSecret.Data[rancherCcRegionField], "Expected region field of cloud credential copy to not match updated cloud credential secret")
}

// TestUpdateOCNEclusterCloudCreds tests checkClusterCredentials
// GIVEN an updated cloud credential secret
// WHEN CheckClusterCredentials is called
// THEN the a secret should be created to match the new cloud credential, if one doesn't already exist
func TestCheckClusterCredentialsNoCopyOfSecret(t *testing.T) {
	scheme := k8scheme.Scheme
	_ = vzapi.AddToScheme(scheme)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(scheme, newClusterRepoResources()...)
	gvr := GetOCNEClusterAPIGVRForResource("clusters")
	// add dynamic elements to scheme
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: gvr.Group, Version: gvr.Version, Kind: "Cluster" + "List"}, &unstructured.Unstructured{})
	ccSecretName := "test-secret"
	ccSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ccSecretName,
			Namespace: rancher.CattleGlobalDataNamespace,
		},
		Data: map[string][]byte{
			rancherCcFingerprintField: []byte("fingerprint-new"),
			rancherCcTenancyField:     []byte("test-tenancy-new"),
			rancherCcRegionField:      []byte("test-region-new"),
		},
	}

	vz := &vzapi.Verrazzano{}

	fakeClient := fakes.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(ccSecret, vz).Build()
	r := &VerrazzanoSecretsReconciler{
		Client:        fakeClient,
		DynamicClient: dynamicClient,
		Scheme:        scheme,
		StatusUpdater: nil,
	}

	err := r.checkClusterCredentials(ccSecret)
	assert.NoError(t, err)
	updatedClusterSecretCopy := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: "cluster", Name: "cluster-principal"}, updatedClusterSecretCopy)
	assert.NoError(t, err)
	assert.Equalf(t, updatedClusterSecretCopy.Data[ociCapiFingerprintField], ccSecret.Data[rancherCcFingerprintField], "Expected fingerprint field of cloud credential copy to match updated cloud credential secret")
	assert.Equalf(t, updatedClusterSecretCopy.Data[ociCapiTenancyField], ccSecret.Data[rancherCcTenancyField], "Expected tenancy field of cloud credential copy to match updated cloud credential secret")
	assert.NotEqualf(t, updatedClusterSecretCopy.Data[ociCapiRegionField], ccSecret.Data[rancherCcRegionField], "Expected region field of cloud credential copy to not match updated cloud credential secret")
}

// newClusterRepoResources creates resources that will be loaded into the dynamic k8s client
func newClusterRepoResources() []runtime.Object {
	ocneCluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "cluster",
			},
			"spec": map[string]interface{}{
				"genericEngineConfig": map[string]interface{}{
					"cloudCredentialId": "cattle-global-data:test-secret",
				},
			},
		},
	}
	gvk := schema.GroupVersionKind{
		Group:   "management.cattle.io",
		Version: "v3",
		Kind:    "Cluster",
	}
	ocneCluster.SetGroupVersionKind(gvk)
	return []runtime.Object{ocneCluster}
}
