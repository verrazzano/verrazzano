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
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
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
	clusterName1 = "test-cluster-1"
	clusterName2 = "test-cluster-2"
	clusterName3 = "test-cluster-3"

	clusterID1 = "c-6fw7h"
	clusterID2 = "c-2rd1k"
	clusterID3 = "c-9gx3y"

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
	expectHTTPCalls(httpMock, false)

	// call the syncer
	r := &RancherClusterSyncer{Client: k8sFake}
	log := r.initLogger()
	r.syncRancherClusters(log)

	mocker.Finish()

	// we should have created two VMCs
	// note that the VMCs we create are named using the Rancher cluster id
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: clusterID1, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID1, cr.Status.RancherRegistration.ClusterID)

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterID2, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID2, cr.Status.RancherRegistration.ClusterID)

	// the pre-existing VMC that is not labeled (so not auto-created) should still be here
	err = r.Get(context.TODO(), types.NamespacedName{Name: preExistingClusterNotLabeled, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)

	// the pre-existing VMC that is labeled (so auto-created) should have been deleted
	err = r.Get(context.TODO(), types.NamespacedName{Name: preExistingClusterLabeled, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.True(errors.IsNotFound(err))
}

// TestSyncRancherClustersWithPaging tests the SyncRancherClusters function with Rancher API paging.
// GIVEN clusters that exist in Rancher that do not have VMCs
//  WHEN the SyncRancherClusters function is called
//   AND the Rancher API call to retrieve clusters results in multiple pages of clusters
//  THEN VMCs are created for clusters returned in all pages
func TestSyncRancherClustersWithPaging(t *testing.T) {
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
	expectHTTPCalls(httpMock, true)

	// call the syncer
	r := &RancherClusterSyncer{Client: k8sFake}
	log := r.initLogger()
	r.syncRancherClusters(log)

	mocker.Finish()

	// we should have created three VMCs (2 from the first page of the clusters API response and one from the 2nd)
	// note that the VMCs we create are named using the Rancher cluster id
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: clusterID1, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID1, cr.Status.RancherRegistration.ClusterID)

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterID2, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID2, cr.Status.RancherRegistration.ClusterID)

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterID3, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID3, cr.Status.RancherRegistration.ClusterID)
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
func expectHTTPCalls(httpMock *mocks.MockRequestSender, testPaging bool) {
	// expect an HTTP request to fetch the admin token from Rancher
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(strings.Split(loginPath, "?")[0])).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			r := ioutil.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
			resp := &http.Response{
				StatusCode: http.StatusCreated,
				Body:       r,
				Request:    &http.Request{Method: http.MethodPost},
			}
			return resp, nil
		})

	// expect an HTTP request to fetch the clusters from Rancher
	httpMock.EXPECT().
		Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clustersPath)).
		DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
			var r io.ReadCloser
			if testPaging {
				// if we're testing paging, include a next page URL
				data := `{"pagination":{"next":"https://host` + clustersPath + `?token=test"}, "data":[{"name":"` +
					clusterName1 + `","id":"` + clusterID1 + `"},{"name":"` + clusterName2 + `","id":"` + clusterID2 + `"}]}`
				r = ioutil.NopCloser(bytes.NewReader([]byte(data)))
			} else {
				data := `{"data":[{"name":"` + clusterName1 + `","id":"` + clusterID1 + `"},{"name":"` + clusterName2 + `","id":"` + clusterID2 + `"}]}`
				r = ioutil.NopCloser(bytes.NewReader([]byte(data)))
			}
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       r,
				Request:    &http.Request{Method: http.MethodGet},
			}
			return resp, nil
		})

	// if we're testing paging, return another page with a cluster
	if testPaging {
		httpMock.EXPECT().
			Do(gomock.Not(gomock.Nil()), mockmatchers.MatchesURI(clustersPath)).
			DoAndReturn(func(httpClient *http.Client, req *http.Request) (*http.Response, error) {
				data := `{"data":[{"name":"` + clusterName3 + `","id":"` + clusterID3 + `"}]}`
				r := ioutil.NopCloser(bytes.NewReader([]byte(data)))
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Request:    &http.Request{Method: http.MethodGet},
				}
				return resp, nil
			})
	}
}
