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

const (
	waitTimeout          = 30 * time.Minute
	pollingInterval      = 1 * time.Minute
)

var (
	t = framework.NewTestFramework("capi-ocne-driver")
	httpClient *retryablehttp.Client
	rancherURL string
	adminToken string
	cloudCredentialId string
)

var beforeSuite = t.BeforeSuiteFunc(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	httpClient = pkg.EventuallyVerrazzanoRetryableHTTPClient()
	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherURL = pkg.EventuallyGetURLForIngress(t.Logs, api, "cattle-system", "rancher", "https")
	adminToken = pkg.GetRancherAdminToken(t.Logs, httpClient, rancherURL)
	cloudCredentialId = createCloudCredential("testing-creds")
})
var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("OCNE Cluster Driver", Label("TODO: appropriate label"), Serial, func() {
	t.Context("Cluster Creation", func() {
		t.It("creates an active cluster", func() {
			// Create the cluster
			clusterName := "strudel"
			createCluster(clusterName)

			// Verify the cluster is active
			Eventually(func() bool {return clusterIsActive(clusterName)}, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("Cluster %s is not active", clusterName))
		})
	})
})

func createCloudCredential(credentialName string) string {
	requestURL := rancherURL + "/v3/cloudcredentials"
	requestBody := []byte(fmt.Sprintf(`{
		"_name": "%s",
		"_type": "provisioning.cattle.io/cloud-credential",
		"type": "provisioning.cattle.io/cloud-credential",
		"name": "%s",
		"description": "dummy description",
		"metadata": {
			"generateName": "cc-",
			"namespace": "fleet-default"
		},
		"annotations": {
			"provisioning.cattle.io/driver": "oracle"
		},
		"ocicredentialConfig": {
			"fingerprint": "<<FILL ME>>",
			"privateKeyContents": "<<FILL ME>>",
			"tenancyId": "<<FILL ME>>",
			"userId": "<<FILL ME>>"
		}
	}`, credentialName, credentialName))

	// FIXME: add error checking
	request, _ := retryablehttp.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	response, _ := httpClient.Do(request)
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	jsonBody, _ := gabs.ParseJSON(body)
	credId := fmt.Sprint(jsonBody.Path("id").Data())
	return credId
}

// Creates an OCNE cluster through CAPI
func createCluster(clusterName string) {
	requestURL := rancherURL + "/v3/cluster"
	// FIXME: which vcn, compartment, etc. to use
	requestBody := []byte(fmt.Sprintf(`{
		"description": "testing cluster",
		"name": "%s",
		"ociocneEngineConfig": {
			"displayName": "%s",
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
			"cloudCredentialId": "%s",
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
	}`, clusterName, clusterName, cloudCredentialId))
	
	// FIXME: add error checking
	request, _ := retryablehttp.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(requestBody))
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	httpClient.Do(request)
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