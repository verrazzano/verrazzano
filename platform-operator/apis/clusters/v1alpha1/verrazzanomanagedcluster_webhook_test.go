// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateWithSecretAndConfigMap tests the validation of a valid VerrazzanoManagedCluster secret and valid verrazzano-admin-cluster configmap
// GIVEN a call validate VerrazzanoManagedCluster
// WHEN the VerrazzanoManagedCluster has valid secret specified and verrazzano-admin-cluster configmap is valid
// THEN the validation should succeed
func TestCreateWithSecretAndConfigMap(t *testing.T) {
	const secretName = "mySecret"

	// fake client needed to get secret
	getClientFunc = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme(),
			&v1alpha1.Verrazzano{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-verrazzano",
					Namespace: "default",
				},
				Status: v1alpha1.VerrazzanoStatus{
					Conditions: []v1alpha1.Condition{
						{
							Type: v1alpha1.InstallComplete,
						},
					},
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      constants.AdminClusterConfigMapName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
				Data: map[string]string{
					constants.ServerDataKey: "https://testUrl",
				},
			}), nil
	}
	defer func() { getClientFunc = getClient }()

	// VMC to be validated
	vz := VerrazzanoManagedCluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testMC",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoManagedClusterSpec{
			PrometheusSecret: secretName,
		},
	}
	err := vz.ValidateCreate()
	assert.NoError(t, err, "Error validating VerrazzanoMultiCluster resource")
}

// TestCreateNoConfigMap tests the validation of missing verrazzano-admin-cluster configmap
// GIVEN a call validate VerrazzanoManagedCluster
// WHEN the verrazzano-admin-cluster configmap doesn't exist
// THEN the validation should fail
func TestCreateNoConfigMap(t *testing.T) {
	const secretName = "mySecret"

	// fake client needed to get secret
	getClientFunc = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(k8scheme.Scheme,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
			}), nil
	}
	defer func() { getClientFunc = getClient }()

	// VMC to be validated
	vz := VerrazzanoManagedCluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testMC",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoManagedClusterSpec{
			PrometheusSecret: secretName,
		},
	}
	err := vz.ValidateCreate()
	assert.EqualError(t, err, "The ConfigMap verrazzano-admin-cluster does not exist in namespace verrazzano-mc",
		"Expected correct error message")
}

// TestCreateWithSecretConfigMapMissingServer tests the validation of verrazzano-admin-cluster configmap with missing server data
// GIVEN a call validate VerrazzanoManagedCluster
// WHEN the verrazzano-admin-cluster configmap is missing server data
// THEN the validation should fail
func TestCreateWithSecretConfigMapMissingServer(t *testing.T) {
	const secretName = "mySecret"

	// fake client needed to get secret
	getClientFunc = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(k8scheme.Scheme,
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
			},
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      constants.AdminClusterConfigMapName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
			}), nil
	}
	defer func() { getClientFunc = getClient }()

	// VMC to be validated
	vz := VerrazzanoManagedCluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testMC",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoManagedClusterSpec{
			PrometheusSecret: secretName,
		},
	}
	err := vz.ValidateCreate()
	assert.EqualError(t, err, "Data with key \"server\" contains invalid url \"\" in the ConfigMap verrazzano-admin-cluster namespace verrazzano-mc",
		"Expected correct error message")
}

// TestCreateMissingSecretName tests the validation of a VerrazzanoManagedCluster with a missing secret name
// GIVEN a call validate VerrazzanoManagedCluster
// WHEN the VerrazzanoManagedCluster is missing the secret name
// THEN the validation should fail
func TestCreateMissingSecretName(t *testing.T) {
	getClientFunc = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(k8scheme.Scheme), nil
	}
	defer func() { getClientFunc = getClient }()
	vz := VerrazzanoManagedCluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	err := vz.ValidateCreate()
	assert.EqualError(t, err, "The name of the Prometheus secret in namespace verrazzano-mc must be specified",
		"Expected correct error message for missing secret")
}

// TestCreateMissingSecret tests the validation of a missing Prometheus secret in the MC namespace
// GIVEN a call validate VerrazzanoManagedCluster
// WHEN the multi-cluster namespace is missing the secret
// THEN the validation should fail
func TestCreateMissingSecret(t *testing.T) {
	const secretName = "mySecret"
	getClientFunc = func() (client.Client, error) {
		return fake.NewFakeClientWithScheme(newScheme()), nil
	}
	defer func() { getClientFunc = getClient }()

	vz := VerrazzanoManagedCluster{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testMC",
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: VerrazzanoManagedClusterSpec{
			PrometheusSecret: secretName,
		},
	}
	err := vz.ValidateCreate()
	assert.EqualError(t, err, "The Prometheus secret mySecret does not exist in namespace verrazzano-mc",
		"Expected correct error message for missing secret")
}
