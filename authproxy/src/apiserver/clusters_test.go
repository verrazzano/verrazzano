// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package apiserver

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const caCertTestData = "../../internal/testdata/test-ca.crt"

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
	apiReq := APIRequest{
		APIServerURL: "https://apiserver.io",
	}
	testClusterName := "testName"
	urlPath, err := url.Parse(fmt.Sprintf("https://apiserver.io/clusters/%s/apidata", testClusterName))
	assert.NoError(t, err)

	req := &http.Request{
		URL: urlPath,
	}

	err = apiReq.reformatClusterPath(req, testClusterName, localClusterPrefix)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s%s/apidata", apiReq.APIServerURL, localClusterPrefix), req.URL.String())
	assert.Equal(t, "apiserver.io", req.Host)
}

// TestGetManagedClusterAPIURL tests that the managed cluster API URL can be obtained from the VMC
func TestGetManagedClusterAPIURL(t *testing.T) {
	vmcName := "testVMC"
	scheme := k8scheme.Scheme
	err := v1alpha1.AddToScheme(scheme)
	assert.NoError(t, err)

	vmc := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmcName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	apiReq := APIRequest{
		Log: zap.S(),
	}

	// GIVEN a cluster name
	// WHEN the API URL is empty
	// THEN an error is returned
	err = apiReq.setManagedClusterAPIURL(vmc)
	assert.Error(t, err)

	testURL := "https://apiserver.io"
	vmc = v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vmcName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			APIUrl: testURL,
		},
	}

	// GIVEN a cluster name
	// WHEN the VMC exists and has an API URL
	// THEN the API URL is returned
	err = apiReq.setManagedClusterAPIURL(vmc)
	assert.NoError(t, err)
	assert.Equal(t, testURL, apiReq.APIServerURL)
}

// TestRewriteClientCACerts tests that the CA certs can be rewritten in the client for managed cluster requests
func TestRewriteClientCACerts(t *testing.T) {
	apiReq := APIRequest{
		Log: zap.S(),
	}
	scheme := k8scheme.Scheme
	err := v1alpha1.AddToScheme(scheme)
	assert.NoError(t, err)

	emptySecret := v1.Secret{}

	secretName := "test-secret"
	vmc := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
		Spec: v1alpha1.VerrazzanoManagedClusterSpec{
			CASecret: secretName,
		},
	}

	secretData, err := os.ReadFile(caCertTestData)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		vmc       v1alpha1.VerrazzanoManagedCluster
		secret    *v1.Secret
		expectErr bool
	}{
		// GIVEN a request to update the client CA
		// WHEN the vmc does not have the CA secret populated
		// THEN an error is returned
		{
			name:      "test empty VMC",
			secret:    &emptySecret,
			expectErr: true,
		},
		// GIVEN a request to update the client CA
		// WHEN the CA secret does not exist
		// THEN an error is returned
		{
			name:      "test secret does not exist",
			vmc:       vmc,
			secret:    &emptySecret,
			expectErr: true,
		},
		// GIVEN a request to update the client CA
		// WHEN the CA data is empty
		// THEN an error is returned
		{
			name: "test secret data empty",
			vmc:  vmc,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
			},
			expectErr: true,
		},
		// GIVEN a request to update the client CA
		// WHEN the CA data is populated
		// THEN no error is returned
		{
			name: "test valid secret",
			vmc:  vmc,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: constants.VerrazzanoMultiClusterNamespace,
				},
				Data: map[string][]byte{
					caCertKey: secretData,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiReq.K8sClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.secret).Build()

			err := apiReq.rewriteClientCACerts(tt.vmc)
			if tt.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
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

	secretName := "test-secret"
	secretData, err := os.ReadFile(caCertTestData)
	assert.NoError(t, err)

	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmcName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Spec: v1alpha1.VerrazzanoManagedClusterSpec{
				CASecret: secretName,
			},
			Status: v1alpha1.VerrazzanoManagedClusterStatus{
				APIUrl: testURL,
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Data: map[string][]byte{
				caCertKey: secretData,
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

// TestProcessManagedClusterResources tests that the managed cluster resources can be properly processed before the
// request is sent out
// GIVEN a request to process the managed cluster resources
// WHEN the VMC is processed
// THEN no error is returned and the API resource is updated
func TestProcessManagedClusterResources(t *testing.T) {
	vmcName := "testVMC"
	scheme := k8scheme.Scheme
	err := v1alpha1.AddToScheme(scheme)
	assert.NoError(t, err)
	testURL := "https://managed.io"

	secretName := "test-secret"
	secretData, err := os.ReadFile(caCertTestData)
	assert.NoError(t, err)

	cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vmcName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Spec: v1alpha1.VerrazzanoManagedClusterSpec{
				CASecret: secretName,
			},
			Status: v1alpha1.VerrazzanoManagedClusterStatus{
				APIUrl: testURL,
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: constants.VerrazzanoMultiClusterNamespace,
			},
			Data: map[string][]byte{
				caCertKey: secretData,
			},
		},
	).Build()

	apiReq := APIRequest{
		K8sClient: cli,
		Log:       zap.S(),
	}

	err = apiReq.processManagedClusterResources(vmcName)
	assert.NoError(t, err)
	assert.Equal(t, testURL, apiReq.APIServerURL)
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
