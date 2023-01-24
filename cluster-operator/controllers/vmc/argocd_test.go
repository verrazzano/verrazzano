// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconstants "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestCreateArgoCDRequest tests the creation of a ArgoCDConfig which is used to make Argo CD API call followed by a GET on clusters API
func TestCreateArgoCDRequest(t *testing.T) {
	cli := createTestObjects()
	log := vzlog.DefaultLogger()

	testPath := "/api/v1/clusters"
	testBody := "test-body"

	savedRancherHTTPClient := ArgoCDHTTPClient
	defer func() {
		ArgoCDHTTPClient = savedRancherHTTPClient
	}()

	savedRetry := DefaultRetry
	defer func() {
		DefaultRetry = savedRetry
	}()
	DefaultRetry = wait.Backoff{
		Steps:    1,
		Duration: 1 * time.Millisecond,
		Factor:   1.0,
		Jitter:   0.1,
	}

	mocker := gomock.NewController(t)
	httpMock := mocks.NewMockRequestSender(mocker)
	httpMock = expectHTTPRequests(httpMock, testPath, testBody)
	ArgoCDHTTPClient = httpMock

	//Test with the default argocd admin user
	ac, err := newArgoCDConfig(cli, log)
	assert.NoError(t, err)
	response, body, err := sendHTTPRequest(http.MethodGet, testPath, map[string]string{}, "", ac, log)
	assert.NoError(t, err)
	assert.NotNil(t, body)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func createTestObjects() client.WithWatch {
	return fake.NewClientBuilder().WithRuntimeObjects(
		&networkv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vpoconstants.ArgoCDNamespace,
				Name:      vpoconstants.ArgoCDIngress,
			},
			Spec: networkv1.IngressSpec{
				Rules: []networkv1.IngressRule{
					{
						Host: "test-argo.com",
					},
				},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: vpoconstants.ArgoCDNamespace,
				Name:      "argocd-initial-admin-secret",
			},
			Data: map[string][]byte{
				"password": []byte("foo"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.RancherSystemNamespace,
				Name:      "rancher-admin-secret",
			},
			Data: map[string][]byte{
				"password": []byte("bar"),
			},
		}).Build()
}

func expectHTTPRequests(httpMock *mocks.MockRequestSender, testPath, testBody string) *mocks.MockRequestSender {
	httpMock = addSessionTokenMock(httpMock)
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(testPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(testBody)))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		})
	return httpMock
}

func addSessionTokenMock(httpMock *mocks.MockRequestSender) *mocks.MockRequestSender {
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(sessionPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})
	return httpMock
}

// TestCreateClusterSecret tests the creation of a cluster secret required for cluster registration in Argo CD
func TestCreateClusterSecret(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = vzapi.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = networkv1.AddToScheme(scheme)

	var vz = &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				ArgoCD: &vzapi.ArgoCDComponent{
					Enabled: &trueValue,
				},
				Rancher: &vzapi.RancherComponent{
					Enabled: &trueValue,
				},
			},
		},
	}
	var argoClusterUserSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.ArgoCDClusterRancherName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}
	var rancherIngress = &networkv1.Ingress{
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
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(vz, argoClusterUserSecret, rancherIngress).Build()

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
	rc := &VerrazzanoManagedClusterReconciler{
		Client: fakeClient,
		log:    vzlog.DefaultLogger(),
	}

	mocker := gomock.NewController(t)
	mockRequestSender := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherutil.RancherHTTPClient
	defer func() {
		rancherutil.RancherHTTPClient = savedRancherHTTPClient
	}()
	rancherutil.RancherHTTPClient = mockRequestSender

	mockRequestSender.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(loginURIPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			assert.Equal(t, loginQueryString, req.URL.RawQuery)

			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	err := rc.argocdClusterAdd(vmc, []byte("ca"), "https://rancher-url")
	assert.NoError(t, err)
}

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
