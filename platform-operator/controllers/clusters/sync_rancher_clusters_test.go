// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/test/mockmatchers"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	preExistingClusterNoClusterID   = "test-cluster-no-cluster-id"
	preExistingClusterWithClusterID = "test-cluster-with-cluster-id"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = clustersv1alpha1.AddToScheme(testScheme)
}

// TestSyncRancherClusters tests the SyncRancherClusters function.
// GIVEN clusters that exist in Rancher that do not have VMCs
//
//	 AND VMCs that do not have corresponding clusters in Rancher
//	WHEN the SyncRancherClusters function is called
//	THEN VMCs are created and deleted appropriately so that they are in sync
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
	log := vzlog.DefaultLogger()
	r.syncRancherClusters(log)

	mocker.Finish()

	// we should have created two VMCs
	// note that the VMCs we create are named using the Rancher cluster name
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: clusterName1, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID1, cr.Status.RancherRegistration.ClusterID)

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterName2, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID2, cr.Status.RancherRegistration.ClusterID)

	// the pre-existing VMC that has no cluster id in the status should still be here
	err = r.Get(context.TODO(), types.NamespacedName{Name: preExistingClusterNoClusterID, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)

	// also assert that the pre-existing VMC did not have labels added
	asserts.Empty(cr.Labels)

	// the pre-existing VMC that had a cluster id in the status should have been deleted
	err = r.Get(context.TODO(), types.NamespacedName{Name: preExistingClusterWithClusterID, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.True(errors.IsNotFound(err))
}

// TestSyncRancherClustersWithPaging tests the SyncRancherClusters function with Rancher API paging.
// GIVEN clusters that exist in Rancher that do not have VMCs
//
//	WHEN the SyncRancherClusters function is called
//	 AND the Rancher API call to retrieve clusters results in multiple pages of clusters
//	THEN VMCs are created for clusters returned in all pages
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
	log := vzlog.DefaultLogger()
	r.syncRancherClusters(log)

	mocker.Finish()

	// we should have created three VMCs (2 from the first page of the clusters API response and one from the 2nd)
	// note that the VMCs we create are named using the Rancher cluster name
	cr := &clustersv1alpha1.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: clusterName1, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID1, cr.Status.RancherRegistration.ClusterID)

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterName2, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID2, cr.Status.RancherRegistration.ClusterID)

	err = r.Get(context.TODO(), types.NamespacedName{Name: clusterName3, Namespace: constants.VerrazzanoMultiClusterNamespace}, cr)
	asserts.NoError(err)
	asserts.Equal(createdByVerrazzano, cr.Labels[createdByLabel])
	asserts.Equal("true", cr.Labels[vzconst.VerrazzanoManagedLabelKey])
	asserts.Equal(clusterID3, cr.Status.RancherRegistration.ClusterID)
}

// erroringFakeClient wraps a k8s client and returns an error when Create is called and the createReturnsError flag is set
type erroringFakeClient struct {
	client.Client
}

const tooManyRequestsError = "too many requests"

var createReturnsError = false

// Create optionally returns an error - used to simulate an error creating a resource
func (e *erroringFakeClient) Create(_ context.Context, _ client.Object, _ ...client.CreateOption) error {
	if createReturnsError {
		return errors.NewTooManyRequests(tooManyRequestsError, 0)
	}
	return nil
}

// erroringFakeStatusClient allows us to fake a StatusWriter
type erroringFakeStatusClient struct {
}

// Update of status always returns an error
func (e *erroringFakeStatusClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return errors.NewConflict(schema.GroupResource{}, "", nil)
}

// Path is not used in these tests but is required to be implemented by StatusWriter
func (e *erroringFakeStatusClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

// Status returns the fake StatusWriter
func (e *erroringFakeClient) Status() client.StatusWriter {
	return &erroringFakeStatusClient{}
}

// TestEnsureVMCsWithError tests error conditions in the ensureVMCs function
func TestEnsureVMCsWithError(t *testing.T) {
	asserts := assert.New(t)

	// create the k8s fake populated with resources
	k8sFake := createK8sFake()

	rancherClusters := []rancherCluster{{name: "test", id: "test"}}
	r := &RancherClusterSyncer{Client: &erroringFakeClient{Client: k8sFake}}
	log := vzlog.DefaultLogger()

	// GIVEN a list of Rancher clusters fetched from the Rancher API
	//  WHEN a call is made to ensureVMCs and the k8s client returns an error on a call to Create
	//  THEN the expected error is returned
	createReturnsError = true
	err := r.ensureVMCs(rancherClusters, log)
	asserts.ErrorContains(err, tooManyRequestsError)

	// GIVEN a list of Rancher clusters fetched from the Rancher API
	//  WHEN a call is made to ensureVMCs and the k8s client returns an error updating the status
	//  THEN the expected error is returned
	createReturnsError = false
	err = r.ensureVMCs(rancherClusters, log)
	asserts.ErrorContains(err, "Operation cannot be fulfilled")
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
				Name:      preExistingClusterNoClusterID,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
		},
		&clustersv1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: v1.ObjectMeta{
				Name:      preExistingClusterWithClusterID,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Status: clustersv1alpha1.VerrazzanoManagedClusterStatus{
				RancherRegistration: clustersv1alpha1.RancherRegistration{
					ClusterID: "c-v76wj",
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
			r := io.NopCloser(bytes.NewReader([]byte(`{"token":"unit-test-token"}`)))
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
				r = io.NopCloser(bytes.NewReader([]byte(data)))
			} else {
				data := `{"data":[{"name":"` + clusterName1 + `","id":"` + clusterID1 + `"},{"name":"` + clusterName2 + `","id":"` + clusterID2 + `"},` +
					`{"name": "local", "id": "local"}]}`
				r = io.NopCloser(bytes.NewReader([]byte(data)))
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
				r := io.NopCloser(bytes.NewReader([]byte(data)))
				resp := &http.Response{
					StatusCode: http.StatusOK,
					Body:       r,
					Request:    &http.Request{Method: http.MethodGet},
				}
				return resp, nil
			})
	}
}
