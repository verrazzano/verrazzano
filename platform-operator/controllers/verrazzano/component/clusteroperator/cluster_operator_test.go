// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteroperator

import (
	"bytes"
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	rancherAdminSecret = "rancher-admin-secret" //nolint:gosec //#gosec G101
	testBomFilePath    = "../../testdata/test_bom.json"
)

func TestGetOverrides(t *testing.T) {
	testKey := "test-key"
	testVal := "test-val"
	jsonVal := []byte(fmt.Sprintf("{\"%s\":\"%s\"}", testKey, testVal))

	vzA1CR := &v1alpha1.Verrazzano{}
	vzA1CROverrides := vzA1CR.DeepCopy()
	vzA1CROverrides.Spec.Components.ClusterOperator = &v1alpha1.ClusterOperatorComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	vzB1CR := &v1beta1.Verrazzano{}
	vzB1CROverrides := vzB1CR.DeepCopy()
	vzB1CROverrides.Spec.Components.ClusterOperator = &v1beta1.ClusterOperatorComponent{
		InstallOverrides: v1beta1.InstallOverrides{
			ValueOverrides: []v1beta1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		verrazzanoA1   *v1alpha1.Verrazzano
		verrazzanoB1   *v1beta1.Verrazzano
		expA1Overrides interface{}
		expB1Overrides interface{}
	}{
		{
			name:           "test no overrides",
			verrazzanoA1:   vzA1CR,
			verrazzanoB1:   vzB1CR,
			expA1Overrides: []v1alpha1.Overrides{},
			expB1Overrides: []v1beta1.Overrides{},
		},
		{
			name:           "test v1alpha1 enabled nil",
			verrazzanoA1:   vzA1CROverrides,
			verrazzanoB1:   vzB1CROverrides,
			expA1Overrides: vzA1CROverrides.Spec.Components.ClusterOperator.ValueOverrides,
			expB1Overrides: vzB1CROverrides.Spec.Components.ClusterOperator.ValueOverrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(t, tt.expA1Overrides, NewComponent().GetOverrides(tt.verrazzanoA1))
			asserts.Equal(t, tt.expB1Overrides, NewComponent().GetOverrides(tt.verrazzanoB1))
		})
	}
}

// GIVEN a call to AppendOverrides
// WHEN  the env var for the cluster operator image is set
// THEN  the returned key/value pairs contains the image override
func TestAppendOverrides(t *testing.T) {
	customImage := "myreg.io/myrepo/v8o/verrazzano-cluster-operator-dev:local-20210707002801-b7449154"
	err := os.Setenv(constants.VerrazzanoClusterOperatorImageEnvVar, customImage)
	asserts.NoError(t, err)

	kvs, err := AppendOverrides(nil, "", "", "", nil)
	asserts.NoError(t, err)
	asserts.Len(t, kvs, 1)
	asserts.Equal(t, "image", kvs[0].Key)
	asserts.Equal(t, customImage, kvs[0].Value)

	err = os.Unsetenv(constants.VerrazzanoClusterOperatorImageEnvVar)
	asserts.NoError(t, err)

	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, err = AppendOverrides(nil, "", "", "", nil)
	asserts.NoError(t, err)
	asserts.Len(t, kvs, 1)
	asserts.Equal(t, "image", kvs[0].Key)
	asserts.Equal(t, "ghcr.io/verrazzano/VERRAZZANO_CLUSTER_OPERATOR_IMAGE:VERRAZZANO_CLUSTER_OPERATOR_TAG", kvs[0].Value)
}

// TestPostInstallUpgrade tests the PostInstallUpgrade creation of the RoleTemplate
func TestPostInstallUpgrade(t *testing.T) {
	// clear any cached user auth tokens when the test completes
	defer rancherutil.DeleteStoredTokens()

	clustOpComp := clusterOperatorComponent{}

	cli := createClusterUserTestObjects().WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: vzconst.VerrazzanoClusterRancherName,
			},
		},
	).Build()

	mocker := gomock.NewController(t)
	httpMock := createClusterUserExists(mocks.NewMockRequestSender(mocker), http.StatusOK)

	savedRancherHTTPClient := rancherutil.RancherHTTPClient
	defer func() {
		rancherutil.RancherHTTPClient = savedRancherHTTPClient
	}()
	rancherutil.RancherHTTPClient = httpMock

	savedRetry := rancherutil.DefaultRetry
	defer func() {
		rancherutil.DefaultRetry = savedRetry
	}()
	rancherutil.DefaultRetry = wait.Backoff{
		Steps:    1,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	err := clustOpComp.postInstallUpgrade(spi.NewFakeContext(cli, nil, &v1beta1.Verrazzano{}, false))
	asserts.NoError(t, err)

	// Ensure the resource exists after postInstallUpgrade
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(rancher.GVKRoleTemplate)
	err = cli.Get(context.TODO(), types.NamespacedName{Name: vzconst.VerrazzanoClusterRancherName}, &resource)
	asserts.NoError(t, err)
}

// TestPostInstallUpgradeRancherDisabled tests the PostInstallUpgrade when Rancher is disabled
func TestPostInstallUpgradeRancherDisabled(t *testing.T) {
	clustOpComp := clusterOperatorComponent{}
	falseVal := false

	cli := fake.NewClientBuilder().WithObjects(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: vzconst.VerrazzanoClusterRancherName,
			},
		},
	).Build()
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Rancher: &v1alpha1.RancherComponent{
					Enabled: &falseVal,
				},
			},
		},
	}
	err := clustOpComp.postInstallUpgrade(spi.NewFakeContext(cli, vz, nil, false))
	asserts.NoError(t, err)

	// Ensure the resource does not exist after postInstallUpgrade
	resource := unstructured.Unstructured{}
	resource.SetGroupVersionKind(rancher.GVKRoleTemplate)
	err = cli.Get(context.TODO(), types.NamespacedName{Name: vzconst.VerrazzanoClusterRancherName}, &resource)
	asserts.Error(t, err)
}

// TestCreateVZClusterUser tests the creation of the VZ cluster user through the Rancher API
func TestCreateVZClusterUser(t *testing.T) {
	cli := createClusterUserTestObjects().Build()
	mocker := gomock.NewController(t)

	vz := &v1alpha1.Verrazzano{}

	savedRancherHTTPClient := rancherutil.RancherHTTPClient
	defer func() {
		rancherutil.RancherHTTPClient = savedRancherHTTPClient
	}()

	savedRetry := rancherutil.DefaultRetry
	defer func() {
		rancherutil.DefaultRetry = savedRetry
	}()
	rancherutil.DefaultRetry = wait.Backoff{
		Steps:    1,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	tests := []struct {
		name      string
		mock      *mocks.MockRequestSender
		expectErr bool
	}{
		// GIVEN a call to createVZClusterUser
		// WHEN  the user exists
		// THEN  no error is returned
		{
			name:      "test user exists",
			mock:      createClusterUserExists(mocks.NewMockRequestSender(mocker), http.StatusOK),
			expectErr: false,
		},
		// GIVEN a call to createVZClusterUser
		// WHEN  the user API call fails
		// THEN  an error is returned
		{
			name:      "test fail check users",
			mock:      createClusterUserExists(mocks.NewMockRequestSender(mocker), http.StatusUnauthorized),
			expectErr: true,
		},
		// GIVEN a call to createVZClusterUser
		// WHEN  the user does not exist
		// THEN  a user is created and no error is returned
		{
			name:      "test user does not exist",
			mock:      createClusterUserDoesNotExist(mocks.NewMockRequestSender(mocker), http.StatusCreated),
			expectErr: false,
		},
		// GIVEN a call to createVZClusterUser
		// WHEN  the user creation API call fails
		// THEN  an error is returned
		{
			name:      "test user failed create",
			mock:      createClusterUserDoesNotExist(mocks.NewMockRequestSender(mocker), http.StatusUnauthorized),
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clear any cached user auth tokens when the test completes
			defer rancherutil.DeleteStoredTokens()

			rancherutil.RancherHTTPClient = tt.mock
			err := createVZClusterUser(spi.NewFakeContext(cli, vz, nil, false))
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}

func createClusterUserTestObjects() *fake.ClientBuilder {
	return fake.NewClientBuilder().WithRuntimeObjects(
		&networkv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconst.RancherSystemNamespace,
				Name:      constants.RancherIngress,
			},
			Spec: networkv1.IngressSpec{
				Rules: []networkv1.IngressRule{
					{
						Host: "test-rancher.com",
					},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vzconst.RancherSystemNamespace,
				Name:      rancherAdminSecret,
			},
			Data: map[string][]byte{
				"password": []byte(""),
			},
		})
}

func createClusterUserExists(httpMock *mocks.MockRequestSender, getUserStatus int) *mocks.MockRequestSender {
	httpMock = adminTokenMock(httpMock)
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(usersPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"data":[{"dataexists":"true"}]}`)))
			resp := &http.Response{
				StatusCode: getUserStatus,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})
	return httpMock
}

func createClusterUserDoesNotExist(httpMock *mocks.MockRequestSender, createStatus int) *mocks.MockRequestSender {
	httpMock = adminTokenMock(httpMock)
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodGet, usersPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"data":[]}`)))
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURIMethod(http.MethodPost, usersPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"data":[]}`)))
			resp := &http.Response{
				StatusCode: createStatus,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})
	return httpMock
}

func adminTokenMock(httpMock *mocks.MockRequestSender) *mocks.MockRequestSender {
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI("/v3-public/localProviders/local")).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})
	return httpMock
}

// TestCreateVZClusterUser tests the creation of the VZ cluster user through the Rancher API
func TestCreateVZArgoCDClusterUser(t *testing.T) {

	cli := createClusterUserTestObjects().Build()
	mocker := gomock.NewController(t)

	vz := &v1alpha1.Verrazzano{}

	savedRancherHTTPClient := rancherutil.RancherHTTPClient
	defer func() {
		rancherutil.RancherHTTPClient = savedRancherHTTPClient
	}()

	savedRetry := rancherutil.DefaultRetry
	defer func() {
		rancherutil.DefaultRetry = savedRetry
	}()
	rancherutil.DefaultRetry = wait.Backoff{
		Steps:    1,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	tests := []struct {
		name      string
		mock      *mocks.MockRequestSender
		expectErr bool
	}{
		// GIVEN a call to createVZArgoCDUser
		// WHEN  the user exists
		// THEN  no error is returned
		{
			name:      "test user exists",
			mock:      createClusterUserExists(mocks.NewMockRequestSender(mocker), http.StatusOK),
			expectErr: false,
		},
		// GIVEN a call to createVZArgoCDUser
		// WHEN  the user API call fails
		// THEN  an error is returned
		{
			name:      "test fail check users",
			mock:      createClusterUserExists(mocks.NewMockRequestSender(mocker), http.StatusUnauthorized),
			expectErr: true,
		},
		// GIVEN a call to createVZArgoCDUser
		// WHEN  the user does not exist
		// THEN  a user is created and no error is returned
		{
			name:      "test user does not exist",
			mock:      createClusterUserDoesNotExist(mocks.NewMockRequestSender(mocker), http.StatusCreated),
			expectErr: false,
		},
		// GIVEN a call to createVZArgoCDUser
		// WHEN  the user creation API call fails
		// THEN  an error is returned
		{
			name:      "test user failed create",
			mock:      createClusterUserDoesNotExist(mocks.NewMockRequestSender(mocker), http.StatusUnauthorized),
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clear any cached user auth tokens when the test completes
			defer rancherutil.DeleteStoredTokens()

			rancherutil.RancherHTTPClient = tt.mock
			err := createVZArgoCDUser(spi.NewFakeContext(cli, vz, nil, false))
			if tt.expectErr {
				asserts.Error(t, err)
				return
			}
			asserts.NoError(t, err)
		})
	}
}
