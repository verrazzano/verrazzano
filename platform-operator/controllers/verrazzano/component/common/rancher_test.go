// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
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
//
//	WHEN GetAdminSecret is called with a client that has an admin secret in cluster
//	THEN GetAdminSecret should return the value of the Rancher admin secret
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
//
//	WHEN GetRootCA is called with a client that has an CA secret in cluster
//	THEN GetRootCA should return the value of the Rancher CA secret
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
			spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).WithObjects(&authConfig).Build(), nil, nil, false),
			false,
		},
		{
			"should throw an error when the auth config is not present",
			spi.NewFakeContext(fake.NewClientBuilder().WithScheme(getScheme()).Build(), nil, nil, false),
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

// TestGetAdditionalCA tests the GetAdditionalCA function
// GIVEN a client and secret doesn't exist
// WHEN  the GetAdditionalCA function is called
// THEN  the function call fails and returns empty slice
func TestGetAdditionalCA(t *testing.T) {
	cli := fake.NewClientBuilder().WithScheme(getScheme()).WithObjects().Build()
	caKey := GetAdditionalCA(cli)
	assert.Equal(t, []byte{}, caKey)

}

// TestCertPool tests the CertPool function
// GIVEN a component context
// WHEN  the CertPool function is called
// THEN  the function call succeeds and new certpool is returned
func TestCertPool(t *testing.T) {
	certs := []byte{}
	certPool := CertPool(certs)
	assert.NotNil(t, certPool)

}

// TestRetry tests Retry for a function
// GIVEN a retryable function
//
//	WHEN Retry is called with retryOnError as false and function returns error
//	THEN Retry should return the same error and not retry
//	WHEN Retry is called function does not returns error
//	THEN Retry should retry the function until timeout occurs
//	WHEN Retry is called with retryOnError as true and function returns error
//	THEN Retry should retry the function until timeout and return the last error
//	WHEN Retry is called and function return true with no error
//	THEN Retry should return no error
func TestRetry(t *testing.T) {
	var tests = []struct {
		testName       string
		retryable      wait.ConditionFunc
		backoff        wait.Backoff
		retryOnError   bool
		isError        bool
		isTimeoutError bool
	}{
		{
			"should return error when retryOnError is false and function returns error",
			func() (bool, error) { return false, fmt.Errorf("") },
			wait.Backoff{Steps: 1, Duration: 2 * time.Second},
			false,
			true,
			false,
		},
		{
			"should retry until timeout when function does not return error and return timeout error",
			func() (bool, error) { return false, nil },
			wait.Backoff{Steps: 1, Duration: 2 * time.Second},
			false,
			true,
			true,
		},
		{
			"should retry until timeout when function return error but retryOnError is true and eventually return the error returned by function",
			func() (bool, error) { return false, fmt.Errorf("") },
			wait.Backoff{Steps: 1, Duration: 2 * time.Second},
			true,
			true,
			false,
		},
		{
			"should return no error when condition is met",
			func() (bool, error) { return true, nil },
			wait.Backoff{Steps: 1, Duration: 2 * time.Second},
			true,
			false,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			err := Retry(tt.backoff, vzlog.DefaultLogger(), false, tt.retryable)
			if tt.isError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				if tt.isTimeoutError {
					assert.Equal(t, wait.ErrWaitTimeout.Error(), err)
				} else {
					assert.NotEqual(t, wait.ErrWaitTimeout.Error(), err)
				}
			}
		})
	}
}
