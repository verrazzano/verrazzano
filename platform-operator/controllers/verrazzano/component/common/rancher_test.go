// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func getScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = networking.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = v1.AddToScheme(scheme)
	return scheme
}

func createRootCASecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: CattleSystem,
			Name:      RancherIngressCAName,
		},
		Data: map[string][]byte{
			RancherCACert: []byte("blahblah"),
		},
	}
}

func createAdminSecret() v1.Secret {
	return v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: CattleSystem,
			Name:      RancherAdminSecret,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
}

func createKeycloakAuthConfig() unstructured.Unstructured {
	authConfig := unstructured.Unstructured{
		Object: map[string]interface{}{},
	}
	authConfig.SetGroupVersionKind(GVKAuthConfig)
	authConfig.SetName(AuthConfigKeycloak)
	return authConfig
}

// TestGetAdminSecret tests retrieving the Rancher admin secret value
// GIVEN a client
//  WHEN GetAdminSecret is called with a client that has an admin secret in cluster
//  THEN GetAdminSecret should return the value of the Rancher admin secret
func TestGetAdminSecret(t *testing.T) {
	secret := createAdminSecret()
	var tests = []struct {
		testName string
		c        client.Client
		isError  bool
	}{
		{
			"should retrieve the secret when it exists",
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&secret).Build(),
			false,
		},
		{
			"should throw an error when the secret is not present",
			fake.NewClientBuilder().WithScheme(getScheme()).Build(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			_, err := GetAdminSecret(tt.c)
			if tt.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

// TestGetRancherTLSRootCA tests retrieving the Rancher CA certificate
// GIVEN a client
//  WHEN GetRootCA is called with a client that has an CA secret in cluster
//  THEN GetRootCA should return the value of the Rancher CA secret
func TestGetRancherTLSRootCA(t *testing.T) {
	secret := createRootCASecret()
	var tests = []struct {
		testName string
		c        client.Client
		notFound bool
	}{
		{
			"should retrieve the secret when it exists",
			fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&secret).Build(),
			false,
		},
		{
			"should throw an error when the secret is not present",
			fake.NewClientBuilder().WithScheme(getScheme()).Build(),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			cert, err := GetRootCA(tt.c)
			assert.Nil(t, err)
			if tt.notFound {
				assert.Nil(t, cert)
			} else {
				assert.NotNil(t, cert)
			}
		})
	}
}

func TestUpdateKeycloakOIDCAuthConfig(t *testing.T) {
	authConfig := createKeycloakAuthConfig()
	var tests = []struct {
		testName string
		ctx      spi.ComponentContext
		isError  bool
	}{
		{
			"should update the auth config when it exists",
			spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&authConfig).Build(), nil, false),
			false,
		},
		{
			"should throw an error when the auth config is not present",
			spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), nil, false),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := UpdateKeycloakOIDCAuthConfig(tt.ctx, map[string]interface{}{})
			if !tt.isError {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestGetRancherMgmtApiGVKForKind(t *testing.T) {
	kind := GetRancherMgmtAPIGVKForKind("kind")
	assert.Equal(t, APIGroupRancherManagement, kind.Group)
	assert.Equal(t, APIGroupVersionRancherManagement, kind.Version)
	assert.Equal(t, "kind", kind.Kind)

}
