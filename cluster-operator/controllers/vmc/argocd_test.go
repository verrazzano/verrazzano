// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	tokensPath = "/v3/tokens"
)

// TestMutateSecret tests the creation of a cluster secret required for cluster registration in Argo CD
// GIVEN a call to mutate secret
//
//	WHEN the secret does not contain created/expiresAt labels
//	THEN we get a new token followed by created/expiresAt labels are set in the secret
func TestCreateClusterSecret(t *testing.T) {
	cli := generateClientObject()
	log := vzlog.DefaultLogger()

	getBody := "{\"created\":\"xxx\", \"expiresAt\": \"yyy\"}"
	postBody := "{\"token\":\"xxx\", \"name\": \"testToken\"}"

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
			Namespace: rancherNamespace,
			Name:      "cluster",
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			RancherRegistration: v1alpha1.RancherRegistration{
				ClusterID: "cluster-id",
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
			Annotations: map[string]string{"created": "xxx", "expiresAt": "yyy"},
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
	clusterID := "foo"
	rancherURL := "https://rancher-url"
	caData := []byte("bar")

	mocker := gomock.NewController(t)
	httpMock := mocks.NewMockRequestSender(mocker)
	httpMock = expectHTTPRequests(httpMock, getBody, postBody)
	rancherutil.RancherHTTPClient = httpMock

	rc, err := rancherutil.NewAdminRancherConfig(cli, log)
	assert.NoError(t, err)

	err = r.mutateClusterSecret(secret, rc, vmc.Name, clusterID, rancherURL, caData)
	assert.NoError(t, err)
	assert.NotNil(t, secret.Annotations["verrazzano.io/createTimestamp"])
	assert.NotNil(t, secret.Annotations["verrazzano.io/expiresAtTimestamp"])
}

func expectHTTPRequests(httpMock *mocks.MockRequestSender, getBody, postBody string) *mocks.MockRequestSender {
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(tokensPath+"/testToken")).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(getBody)))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		}).Times(1)
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(tokensPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(postBody)))
			resp = &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
			}
			return resp, nil
		}).Times(1)
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
		}).Times(1)
	return httpMock
}

func generateClientObject() client.WithWatch {
	return fake.NewClientBuilder().WithRuntimeObjects(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "demo" + "-" + clusterSecretName,
				Namespace: constants.ArgoCDNamespace,
			},
			Data: map[string][]byte{
				"password": []byte("foobar"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rancher-admin-secret",
				Namespace: constants.RancherSystemNamespace,
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
func TestUpdateArgoCDClusterRoleBindingTemplate(t *testing.T) {
	a := assert.New(t)

	vmcNoID := &v1alpha1.VerrazzanoManagedCluster{}

	clusterID := "testID"
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
