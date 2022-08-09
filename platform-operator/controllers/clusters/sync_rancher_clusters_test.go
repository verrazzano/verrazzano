// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterName1                 = "test-cluster-1"
	clusterName2                 = "test-cluster-2"
	preExistingClusterNotLabeled = "test-cluster-not-labeled"
	preExistingClusterLabeled    = "test-cluster-labeled"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)
}

// TestSyncRancherClusters tests the SyncRancherClusters function.
// GIVEN clusters that exist in Rancher that do not have VMCs
//   AND VMCs that do not have corresponding clusters in Rancher
//  WHEN the SyncRancherClusters function is called
//  THEN VMCs are created and deleted appropriately so that they are in sync
func TestSyncRancherClusters(t *testing.T) {
	asserts := assert.New(t)

	// create mocks and set the Rancher HTTP client to use the HTTP mock
	mocker := gomock.NewController(t)
	httpMock := mocks.NewMockRequestSender(mocker)
	savedRancherHTTPClient := rancherHTTPClient
	defer func() {
		rancherHTTPClient = savedRancherHTTPClient
	}()
	rancherHTTPClient = httpMock

	// create the k8s fake populated with resources
	k8sFake := createK8sFake()

	// expect HTTP calls
	expectHTTPCalls(t, httpMock, false)

	// call the syncer
	r := &RancherClusterSyncer{Client: k8sFake}
	log := r.initLogger()
	r.syncRancherClusters(log)

	mocker.Finish()

	// we should have created two VMCs
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: clusterName1, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterName2, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])

	// the pre-existing VMC that is not labeled (so not auto-created) should still be here
	err = r.Get(context.TODO(), types.NamespacedName{Name: preExistingClusterNotLabeled, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)

	// the pre-existing VMC that is labeled (so auto-created) should have been deleted
	err = r.Get(context.TODO(), types.NamespacedName{Name: preExistingClusterLabeled, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.True(errors.IsNotFound(err))
}

// createK8sFake creates a k8s fake populated with resources for testing
func createK8sFake() client.Client {
	return fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
		&corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherAdminSecret,
			},
		},
		&k8net.Ingress{
			ObjectMeta: v1.ObjectMeta{
				Namespace: rancherNamespace,
				Name:      rancherIngressName,
			},
			Spec: k8net.IngressSpec{
				Rules: []k8net.IngressRule{
					{
						Host: "rancher-ingress",
					},
				},
			},
		},
		&clustersv1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      preExistingClusterNotLabeled,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
		},
		&clustersv1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      preExistingClusterLabeled,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
				Labels: map[string]string{
					createdByLabel: createdByVerrazzano,
				},
			},
		}).Build()
}

// expectHTTPCalls mocks the HTTP calls we expect the Rancher client to make
func expectHTTPCalls(t *testing.T, httpMock *mocks.MockRequestSender, testPaging bool) {
	asserts := assert.New(t)

	// expect an HTTP request to fetch the admin token from Rancher
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			urlParts := strings.Split(loginPath, "?")
			asserts.Equal(urlParts[0], req.URL.Path)
			asserts.Equal(urlParts[1], req.URL.RawQuery)

			r := ioutil.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// expect an HTTP request to fetch all of the clusters from Rancher
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			asserts.Equal(clustersPath, req.URL.Path)

			var r io.ReadCloser
			if testPaging {
				r = ioutil.NopCloser(bytes.NewReader([]byte(`{"pagination":{"next":"https://test?token=test"}, "data":[{"name":"` + clusterName1 + `"},{"name":"` + clusterName2 + `"}]}`)))
			} else {
				r = ioutil.NopCloser(bytes.NewReader([]byte(`{"data":[{"name":"` + clusterName1 + `"},{"name":"` + clusterName2 + `"}]}`)))
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})
}
