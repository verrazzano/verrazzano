// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	pkgconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestPushManifestObjects tests the push of manifest objects to a managed cluster
// GIVEN a call to push manifest objects
//
//	WHEN the status of the VMC does not contain the condition update
//	THEN the manifest objects should get pushed to the managed cluster
func TestPushManifestObjects(t *testing.T) {
	a := asserts.New(t)
	c := generateClientObjects()

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
		Client: c,
		log:    vzlog.DefaultLogger(),
	}

	statusTrueVMC := vmc.DeepCopy()
	statusTrueVMC.Status.Conditions = append(statusTrueVMC.Status.Conditions, v1alpha1.Condition{
		Type:   v1alpha1.ConditionManifestPushed,
		Status: corev1.ConditionTrue,
	})

	tests := []struct {
		name       string
		vmc        *v1alpha1.VerrazzanoManagedCluster
		updated    bool
		active     bool
		vzNsExists bool
		mockerFunc func(*mocks.MockRequestSender, *asserts.Assertions, *v1alpha1.VerrazzanoManagedCluster, *VerrazzanoManagedClusterReconciler, string) *mocks.MockRequestSender
		mock       *mocks.MockRequestSender
	}{
		{
			name:       "test not active",
			vmc:        vmc,
			updated:    false,
			active:     false,
			vzNsExists: false,
		},
		{
			name:       "test no verrazzano-system namespace",
			vmc:        vmc,
			updated:    false,
			active:     true,
			vzNsExists: false,
		},
		{
			name:       "test active",
			vmc:        vmc,
			updated:    true,
			active:     true,
			vzNsExists: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// clear any cached user auth tokens when the test completes
			defer rancherutil.DeleteStoredTokens()
			mocker := gomock.NewController(t)
			sender := addManagedClusterMocks(mocker, tt.active, tt.vzNsExists, a, vmc, r)
			rancherutil.RancherHTTPClient = sender
			updated, err := r.pushManifestObjects(context.TODO(), true, tt.vmc)
			a.Equal(tt.updated, updated)
			a.NoError(err)
			mocker.Finish()
		})
	}
}

func addManagedClusterMocks(mocker *gomock.Controller, clusterIsActive bool, vzNsExists bool, a *asserts.Assertions, vmc *v1alpha1.VerrazzanoManagedCluster, r *VerrazzanoManagedClusterReconciler) *mocks.MockRequestSender {
	sender := mocks.NewMockRequestSender(mocker)
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if clusterIsActive && vzNsExists {
		return addActiveClusterNSExistsMock(sender, a, vmc, r, clusterID)
	}
	if !clusterIsActive {
		return addInactiveClusterMock(sender, clusterID)
	}
	return addNoVerrazzanoSystemNSMock(sender, a, vmc, r, clusterID)
}

func generateClientObjects() client.WithWatch {
	return fake.NewClientBuilder().WithRuntimeObjects(
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
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherTLSSecret,
			},
			Data: map[string][]byte{
				"ca.crt": []byte(""),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.VerrazzanoMultiClusterNamespace,
				Name:      pkgconst.VerrazzanoClusterRancherName,
			},
			Data: map[string][]byte{
				"password": []byte(""),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      GetAgentSecretName("cluster"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      GetRegistrationSecretName("cluster"),
			},
		},
		&networkv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rancherNamespace,
				Namespace: rancherIngressName,
			},
			Spec: networkv1.IngressSpec{Rules: []networkv1.IngressRule{{Host: "rancher.unit-test.com"}}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherAdminSecret,
			},
			Data: map[string][]byte{
				"password": []byte("super-secret"),
			},
		},
	).Build()
}

func addInactiveClusterMock(httpMock *mocks.MockRequestSender, clusterID string) *mocks.MockRequestSender {
	addTokenMock(httpMock)

	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clustersPath+"/"+clusterID)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(`{"state":"inactive", "agentImage":"test"}`)))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		}).Times(2)
	return httpMock
}

func addNoVerrazzanoSystemNSMock(httpMock *mocks.MockRequestSender, a *asserts.Assertions, vmc *v1alpha1.VerrazzanoManagedCluster, r *VerrazzanoManagedClusterReconciler, clusterID string) *mocks.MockRequestSender {
	addTokenMock(httpMock)
	addActiveClusterMock(httpMock, 2)
	addVerrazzanoSystemNamespaceMock(httpMock, clusterID, false)
	return httpMock
}

func addActiveClusterNSExistsMock(httpMock *mocks.MockRequestSender, a *asserts.Assertions, vmc *v1alpha1.VerrazzanoManagedCluster, r *VerrazzanoManagedClusterReconciler, clusterID string) *mocks.MockRequestSender {
	addActiveClusterMock(httpMock, 2)
	addVerrazzanoSystemNamespaceMock(httpMock, clusterID, true)
	agentSecret, err := r.getSecret(vmc.Namespace, GetAgentSecretName(vmc.Name), true)
	a.NoError(err)
	agentSecret.Namespace = constants.VerrazzanoSystemNamespace
	agentSecret.Name = constants.MCAgentSecret
	httpMock = addNotFoundMock(httpMock, &agentSecret, clusterID)

	regSecret, err := r.getSecret(vmc.Namespace, GetRegistrationSecretName(vmc.Name), true)
	a.NoError(err)
	regSecret.Namespace = constants.VerrazzanoSystemNamespace
	regSecret.Name = constants.MCRegistrationSecret
	httpMock = addNotFoundMock(httpMock, &regSecret, clusterID)
	return httpMock
}

func addActiveClusterMock(httpMock *mocks.MockRequestSender, times int) *mocks.MockRequestSender {
	httpMock = addTokenMock(httpMock)
	expectActiveCluster(httpMock, times)
	return httpMock
}

func expectActiveCluster(httpMock *mocks.MockRequestSender, times int) {
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clustersPath+"/"+clusterID)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(`{"state":"active", "agentImage":"test"}`)))
			resp = &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
			}
			return resp, nil
		}).Times(times)
}

func addTokenMock(httpMock *mocks.MockRequestSender) *mocks.MockRequestSender {
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
		}).AnyTimes()
	return httpMock
}

func addVerrazzanoSystemNamespaceMock(httpMock *mocks.MockRequestSender, clusterID string, exists bool) *mocks.MockRequestSender {
	status := http.StatusOK
	if !exists {
		status = http.StatusNotFound
	}
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(k8sClustersPath+clusterID+"/api/v1/namespaces/verrazzano-system")).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var resp *http.Response
			r := io.NopCloser(bytes.NewReader([]byte(`{"kind":"table", "apiVersion":"meta.k8s.io/v1"}`)))
			resp = &http.Response{
				StatusCode: status,
				Body:       r,
			}
			return resp, nil
		})

	return httpMock
}
