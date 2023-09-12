// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestGetClusterName tests that the cluster name can be retrieved from the request path
func TestGetClusterName(t *testing.T) {
	// GIVEN a cluster path
	// WHEN the URL path is parsed
	// THEN the cluster name is returned
	testName := "testName"
	urlPath, err := url.Parse(fmt.Sprintf("https://apiserver.io/clusters/%s/apidata", testName))
	assert.NoError(t, err)

	req := &http.Request{
		URL: urlPath,
	}
	clusterName, err := getClusterName(req)
	assert.NoError(t, err)
	assert.Equal(t, testName, clusterName)

	// GIVEN an empty cluster name
	// WHEN the URL path is parsed
	// THEN an error is returned
	urlPath, err = url.Parse("https://apiserver.io/clusters")
	assert.NoError(t, err)

	req = &http.Request{
		URL: urlPath,
	}
	clusterName, err = getClusterName(req)
	assert.Error(t, err)
}

// TestReformatClusterPath tests that the cluster URL path gets reformatted correctly
// GIVEN a request and path information
// WHEN the request is reformatted
// THEN the new request URL contains the correctly formatted path
func TestReformatClusterPath(t *testing.T) {
	apiReq := APIRequest{}
	testClusterName := "testName"
	urlPath, err := url.Parse(fmt.Sprintf("https://apiserver.io/clusters/%s/apidata", testClusterName))
	assert.NoError(t, err)

	req := &http.Request{
		URL: urlPath,
	}

	apiServerURL := "https://apiserver.io"

	err = apiReq.reformatClusterPath(req, apiServerURL, testClusterName, localClusterPrefix)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s%s/apidata", apiServerURL, localClusterPrefix), req.URL.String())
}

// TestGetManagedClusterAPIURL tests that the managed cluster API URL can be obtained from the VMC
func TestGetManagedClusterAPIURL(t *testing.T) {
	vmcName := "testVMC"
	scheme := k8scheme.Scheme
	err := v1alpha1.AddToScheme(scheme)
	assert.NoError(t, err)

	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmcName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
		},
	).Build()

	apiReq := APIRequest{
		K8sClient: cli,
		Log:       zap.S(),
	}

	// GIVEN a cluster name
	// WHEN the VMC does not exist
	// THEN an error is returned
	_, err = apiReq.getManagedClusterAPIURL("incorrect-name")
	assert.Error(t, err)

	// GIVEN a cluster name
	// WHEN the VMC exists but the API URL is empty
	// THEN an error is returned
	_, err = apiReq.getManagedClusterAPIURL(vmcName)
	assert.Error(t, err)

	testURL := "test-url"
	cli = fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmcName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Status: v1alpha1.VerrazzanoManagedClusterStatus{
				APIUrl: testURL,
			},
		},
	).Build()

	apiReq = APIRequest{
		K8sClient: cli,
		Log:       zap.S(),
	}

	// GIVEN a cluster name
	// WHEN the VMC exists and has an API URL
	// THEN the API URL is returned
	apiURL, err := apiReq.getManagedClusterAPIURL(vmcName)
	assert.NoError(t, err)
	assert.Equal(t, testURL, apiURL)
}

// TestReformatManagedClusterRequest tests that the managed cluster request is reformatted correctly
// GIVEN a managed cluster request
// WHEN the request is processed
// THEN the correct request format is returned
func TestReformatManagedClusterRequest(t *testing.T) {
	vmcName := "testVMC"
	scheme := k8scheme.Scheme
	err := v1alpha1.AddToScheme(scheme)
	assert.NoError(t, err)
	testURL := "https://managed.io"
	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmcName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Status: v1alpha1.VerrazzanoManagedClusterStatus{
				APIUrl: testURL,
			},
		},
	).Build()

	apiReq := APIRequest{
		K8sClient: cli,
		Log:       zap.S(),
	}

	urlPath, err := url.Parse(fmt.Sprintf("https://request.io/clusters/%s/apidata", vmcName))
	req := http.Request{
		URL: urlPath,
	}
	setEmptyToken(&req)

	newReq, err := apiReq.reformatManagedClusterRequest(&req, vmcName)
	assert.NoError(t, err)
	assert.NotNil(t, newReq)
	assert.Equal(t, fmt.Sprintf("%s/clusters/local/apidata", testURL), newReq.URL.String())
}

// TestReformatLocalClusterRequest tests that the local cluster request is formatted correctly
// GIVEN a local cluster request
// WHEN the request is processed
// THEN the correct request is returned
func TestReformatLocalClusterRequest(t *testing.T) {
	apiReq := APIRequest{
		APIServerURL: "https://apiserver.io",
		Log:          zap.S(),
	}

	urlPath, err := url.Parse("https://request.io/clusters/local/apidata")
	req := http.Request{
		URL: urlPath,
	}
	setEmptyToken(&req)

	newReq, err := apiReq.reformatLocalClusterRequest(&req)
	assert.NoError(t, err)
	assert.NotNil(t, newReq)
	assert.Equal(t, "https://apiserver.io/apidata", newReq.URL.String())
}
