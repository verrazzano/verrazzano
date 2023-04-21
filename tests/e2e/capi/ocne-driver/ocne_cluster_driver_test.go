// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	waitTimeout          = 20 * time.Minute
	pollingInterval      = 30 * time.Second

	httpClient *retryablehttp.Client
	rancherURL string
	adminToken string
)

var t = framework.NewTestFramework("install")

var beforeSuite = t.BeforeSuiteFunc(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherURL = pkg.EventuallyGetURLForIngress(t.Logs, api, "cattle-system", "rancher", "https")
	adminToken = pkg.GetRancherAdminToken(t.Logs, httpClient, rancherURL)
	fmt.Println(rancherURL)
})
var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("OCNE Cluster Driver", Label("TODO: appropriate label"), Serial, func() {
	t.Context("Cluster Creation", func() {
		t.It("creates an active cluster", func() {
			clusterName := "capi-ocne-cluster"
			// Create the cluster
			createCluster()

			// Verify the cluster is active
			Eventually(clusterIsActive(clusterName), waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("Cluster %s is not active", clusterName))
		})
	})
})

// Creates an OCNE cluster through CAPI
func createCluster() *http.Response {
	requestURL := rancherURL + "/v3/cluster"
	// FIXME: which vcn, compartment, etc. to use
	requestBody := []byte(`{
		"description": "testing cluster",
		"name": "capi-ocne-cluster",
		"ociocneEngineConfig": {
			"displayName": "capi-ocne-cluster",
			"driverName": "ociocneengine",
			"vcnId": "<<FILL IN>>",
			"nodePublicKeyContents": "<<FILL IN>>",
			"compartmentId": "<<FILL IN>>",
			"workerNodeSubnet": "<<FILL IN>>",
			"controlPlaneSubnet": "<<FILL IN>>",
			"loadBalancerSubnet": "<<FILL IN>>",
			"imageDisplayName": "Oracle-Linux-8.7-2023.01.31-3",
			"kubernetesVersion": "v1.24.8",
			"useNodePvEncryption": true,
			"cloudCredentialId": "<<FILL IN>>",
			"region": "us-ashburn-1",

			"nodeShape": "VM.Standard.E4.Flex",
			"numWorkerNodes": 1,
			"nodeOcpus": 2,
			"nodeMemoryGbs": 32,

			"nodeVolumeGbs": 50,
			"controlPlaneVolumeGbs": 100,

			"podCidr": "<<FILL IN>>",
			"controlPlaneShape": "VM.Standard.E4.Flex",
			"numControlPlaneNodes": 1,
			"controlPlaneMemoryGbs": 16,
			"controlPlaneOcpus": 1
		}
	}`)
	
	// FIXME: add error checking
	request, _ := retryablehttp.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	response, _ := httpClient.Do(request)
	return response
}

// Returns true if the cluster currently exists and is Active
func clusterIsActive(clusterName string) bool {
	// FIXME: add error checking
	jsonBody := getCluster(clusterName)
	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	fmt.Println("State: " + state)
	return state == "active"
}

// Gets a specified cluster by using the Rancher REST API
func getCluster(clusterName string) *gabs.Container {
	// FIXME: add error checking
	requestURL := rancherURL + "/v3/cluster?name=" + clusterName
	request, _ := retryablehttp.NewRequest(http.MethodGet, requestURL, nil)
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	response, _ := httpClient.Do(request)
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	jsonBody, _ := gabs.ParseJSON(body)
	return jsonBody
}