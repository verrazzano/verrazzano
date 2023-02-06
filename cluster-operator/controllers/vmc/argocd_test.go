// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	tokensPath = "/v3/tokens"
	clusterID  = "cluster-id"
	rancherURL = "https://rancher-url"
)

// TestMutateArgoCDClusterSecretWithoutRefresh tests no POST call to obtain new token when we are within 3/4 lifespan of the token
// GIVEN a call to mutateArgCDClusterSecret
//
//	WHEN the secret annotation createTimestamp/expiresAtTimestamp is x(s) and x+4(s) respectively
//	and mutateArgoCDClusterSecret is called immediately
//	THEN we skip obtaining new token
func TestMutateArgoCDClusterSecretWithoutRefresh(t *testing.T) {
	cli := generateClientObject()
	log := vzlog.DefaultLogger()

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

	vmc := &v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoMultiClusterNamespace,
			Name:      "cluster",
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			RancherRegistration: v1alpha1.RancherRegistration{
				ClusterID: clusterID,
			},
		},
	}
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    vzlog.DefaultLogger(),
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "demo" + "-" + clusterSecretName,
			Namespace:   constants.ArgoCDNamespace,
			Annotations: map[string]string{createTimestamp: time.Now().Format(time.RFC3339), expiresAtTimestamp: time.Now().Add(10 * time.Hour).Format(time.RFC3339)},
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}

	mocker := gomock.NewController(t)
	httpMock := mocks.NewMockRequestSender(mocker)
	// Expect an HTTP request to fetch the token from Rancher only
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})
	rancherutil.RancherHTTPClient = httpMock

	caData := []byte("ca")

	rc, err := rancherutil.NewRancherConfigForUser(cli, constants.ArgoCDClusterRancherUsername, "foobar", log)
	assert.NoError(t, err)

	err = r.mutateArgoCDClusterSecret(secret, rc, vmc.Name, clusterID, rancherURL, caData)
	assert.NoError(t, err)

	var rancherConfig ArgoCDRancherConfig
	err = json.Unmarshal([]byte(secret.StringData["config"]), &rancherConfig)
	if err != nil {
		assert.Equal(t, &rancherConfig.BearerToken, "unit-test-token")
	}
}

// TestMutateArgoCDClusterSecretWithRefresh tests POST/GET calls to obtain new token and attrs when we breach 3/4 lifespan of the token
// GIVEN a call to mutateArgoCDClusterSecret
//
//	WHEN the secret annotation createTimestamp/expiresAtTimestamp is x(s) and x+4(s) respectively
//	and we sleep for 4(s)
//	THEN we obtain new token and the annotation createTimestamp/expiresAtTimestamp are updated accordingly
func TestMutateArgoCDClusterSecretWithRefresh(t *testing.T) {
	cli := generateClientObject()
	log := vzlog.DefaultLogger()

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

	vmc := &v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoMultiClusterNamespace,
			Name:      "cluster",
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			RancherRegistration: v1alpha1.RancherRegistration{
				ClusterID: clusterID,
			},
		},
	}
	r := &VerrazzanoManagedClusterReconciler{
		Client: cli,
		log:    vzlog.DefaultLogger(),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "demo" + "-" + clusterSecretName,
			Namespace:   constants.ArgoCDNamespace,
			Annotations: map[string]string{createTimestamp: time.Now().Add(-10 * time.Hour).Format(time.RFC3339), expiresAtTimestamp: time.Now().Format(time.RFC3339)},
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}

	mocker := gomock.NewController(t)
	httpMock := mocks.NewMockRequestSender(mocker)
	httpMock = expectHTTPRequests(httpMock)
	rancherutil.RancherHTTPClient = httpMock

	caData := []byte("ca")

	rc, err := rancherutil.NewRancherConfigForUser(cli, constants.ArgoCDClusterRancherUsername, "foobar", log)
	assert.NoError(t, err)

	err = r.mutateArgoCDClusterSecret(secret, rc, vmc.Name, clusterID, rancherURL, caData)
	assert.NoError(t, err)
	assert.Equal(t, secret.Annotations[createTimestamp], "yyy")
	assert.Equal(t, secret.Annotations[expiresAtTimestamp], "zzz")
}

func expectHTTPRequests(httpMock *mocks.MockRequestSender) *mocks.MockRequestSender {
	// Expect an HTTP request to get created & expiresAt attributes of the token
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(tokensPath+"/testToken")).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte("{\"created\":\"yyy\", \"expiresAt\": \"zzz\"}")))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		})

	// Expect an HTTP request to obtain a new token
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(tokensPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"xxx", "name": "testToken"}`)))
			resp = &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// Expect an HTTP request to fetch the token from Rancher
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
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

func generateClientObject() client.WithWatch {
	return fake.NewClientBuilder().WithRuntimeObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constants.ArgoCDClusterRancherSecretName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Data: map[string][]byte{
				"password": []byte("foobar"),
			},
		},
		&networkv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherIngressName,
			},
			Spec: networkv1.IngressSpec{
				Rules: []networkv1.IngressRule{
					{
						Host: "test-rancher.com",
					},
				},
			},
		},
	).Build()
}

// TestUpdateArgoCDClusterRoleBindingTemplate tests the update of cluster role for 'vz-argocd-reg' user
// GIVEN a call to update argocd cluster role binding
//
//	THEN the template binding are updated accordingly with cluster-owner, cluserID, and userID
/*
func TestUpdateArgoCDClusterRoleBindingTemplate(t *testing.T) {
	a := assert.New(t)

	vmcNoID := &v1alpha1.VerrazzanoManagedCluster{}

	vmcID := vmcNoID.DeepCopy()
	vmcID.Status.RancherRegistration.ClusterID = clusterID

	clusterUserNoData := &unstructured.Unstructured{}
	clusterUserNoData.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    UserKind,
	})
	clusterUserNoData.SetName(constants.ArgoCDClusterRancherUsername)

	clusterUserData := clusterUserNoData.DeepCopy()
	data := clusterUserData.UnstructuredContent()
	data[UserUsernameAttribute] = constants.ArgoCDClusterRancherUsername

	tests := []struct {
		name         string
		vmc          *v1alpha1.VerrazzanoManagedCluster
		expectCreate bool
		expectErr    bool
		user         *unstructured.Unstructured
	}{
		{
			name:         "test nil vmc",
			expectCreate: false,
			expectErr:    false,
			user:         clusterUserData,
		},
		{
			name:         "test vmc no cluster id",
			vmc:          vmcNoID,
			expectCreate: false,
			expectErr:    false,
			user:         clusterUserData,
		},
		{
			name:         "test vmc with cluster id",
			vmc:          vmcID,
			expectCreate: true,
			expectErr:    false,
			user:         clusterUserData,
		},
		{
			name:         "test user doesn't exist",
			vmc:          vmcID,
			expectCreate: false,
			expectErr:    true,
		},
		{
			name:         "test user no username",
			vmc:          vmcID,
			expectCreate: false,
			expectErr:    true,
			user:         clusterUserNoData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := fake.NewClientBuilder()
			if tt.user != nil {
				b = b.WithObjects(tt.user)
			}
			c := b.Build()

			r := &VerrazzanoManagedClusterReconciler{
				Client: c,
				log:    vzlog.DefaultLogger(),
			}
			err := r.updateArgoCDClusterRoleBindingTemplate(tt.vmc)

			if tt.expectErr {
				a.Error(err)
				return
			}
			a.NoError(err)

			if tt.expectCreate {
				name := fmt.Sprintf("crtb-argocd-%s", clusterID)
				resource := &unstructured.Unstructured{}
				resource.SetGroupVersionKind(schema.GroupVersionKind{
					Group:   APIGroupRancherManagement,
					Version: APIGroupVersionRancherManagement,
					Kind:    ClusterRoleTemplateBindingKind,
				})
				err = c.Get(context.TODO(), types.NamespacedName{Namespace: clusterID, Name: name}, resource)
				a.NoError(err)
				a.NotNil(resource)
				a.Equal(clusterID, resource.Object[ClusterRoleTemplateBindingAttributeClusterName])
				a.Equal(constants.ArgoCDClusterRancherUsername, resource.Object[ClusterRoleTemplateBindingAttributeUserName])
				a.Equal("cluster-owner", resource.Object[ClusterRoleTemplateBindingAttributeRoleTemplateName])
			}
		})
	}
}
*/
