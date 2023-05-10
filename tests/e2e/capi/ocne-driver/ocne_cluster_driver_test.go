// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// nolint: gosec // auth constants, not credentials
// gosec: G101: Potential hardcoded credentials
const (
	shortWaitTimeout             = 5 * time.Minute
	shortPollingInterval         = 10 * time.Second
	waitTimeout                  = 30 * time.Minute
	pollingInterval              = 30 * time.Second
	clusterName                  = "strudel"
	createClusterPayloadTemplate = `{
		"description": "testing cluster",
		"name": "{{.ClusterName}}",
		"ociocneEngineConfig": {
			"displayName": "{{.ClusterName}}",
			"driverName": "ociocneengine",
			"vcnId": "{{.VcnID}}",
			"nodePublicKeyContents": "{{.NodePublicKeyContents}}",
			"compartmentId": "{{.CompartmentID}}",
			"workerNodeSubnet": "{{.WorkerNodeSubnet}}",
			"controlPlaneSubnet": "{{.ControlPlaneSubnet}}",
			"loadBalancerSubnet": "{{.LoadBalancerSubnet}}",
			"imageDisplayName": "Oracle-Linux-8.7-2023.02.28-1",
			"kubernetesVersion": "v1.24.8",
			"useNodePvEncryption": true,
			"cloudCredentialId": "{{.CloudCredentialID}}",
			"region": "{{.Region}}",

			"nodeShape": "VM.Standard.E4.Flex",
			"numWorkerNodes": 1,
			"nodeOcpus": 2,
			"nodeMemoryGbs": 32,

			"nodeVolumeGbs": 50,
			"controlPlaneVolumeGbs": 100,

			"podCidr": "{{.PodCIDR}}",
			"controlPlaneShape": "VM.Standard.E4.Flex",
			"numControlPlaneNodes": 1,
			"controlPlaneMemoryGbs": 16,
			"controlPlaneOcpus": 1
		}
	}`
	cloudCredentialsRequestBodyTemplate = `{
		"_name": "{{.CredentialName}}",
		"_type": "provisioning.cattle.io/cloud-credential",
		"type": "provisioning.cattle.io/cloud-credential",
		"name": "{{.CredentialName}}",
		"description": "dummy description",
		"metadata": {
			"generateName": "cc-",
			"namespace": "fleet-default"
		},
		"annotations": {
			"provisioning.cattle.io/driver": "oracle"
		},
		"ocicredentialConfig": {
			"fingerprint": "{{.Fingerprint}}",
			"privateKeyContents": "{{.PrivateKeyContents}}",
			"tenancyId": "{{.TenancyID}}",
			"userId": "{{.UserID}}",
			"region": "{{.Region}}",
			"passphrase": "{{.Passphrase}}"
		}
	}`
)

var (
	t                 = framework.NewTestFramework("capi-ocne-driver")
	httpClient        *retryablehttp.Client
	rancherURL        string
	cloudCredentialID string
)

// cloudCredentialsData needed for template rendering
type cloudCredentialsData struct {
	CredentialName     string
	Fingerprint        string
	PrivateKeyContents string
	TenancyID          string
	UserID             string
	Region             string
	Passphrase         string
}

// capiClusterData needed for template rendering
type capiClusterData struct {
	ClusterName           string
	Region                string
	VcnID                 string
	NodePublicKeyContents string
	CompartmentID         string
	WorkerNodeSubnet      string
	ControlPlaneSubnet    string
	LoadBalancerSubnet    string
	CloudCredentialID     string
	PodCIDR               string
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	//TODO oci get to check it's working

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	if !pkg.IsRancherEnabled(kubeconfigPath) || !pkg.IsCAPIEnabled(kubeconfigPath) {
		Skip("Skipping ocne cluster driver test suite since either of rancher and capi components are not enabled")
	}

	httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed getting http client: %v", err))
	}

	rancherURL, err = helpers.GetRancherURL(t.Logs)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed getting rancherURL: %v", err))
	}
	t.Logs.Infof("rancherURL: %s", rancherURL)

	Eventually(func() error {
		cloudCredentialID, err = createCloudCredential("testing-creds")
		return err
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
})
var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("OCNE Cluster Driver", Label("TODO: appropriate label"), Serial, func() {
	t.Context("OCNE cluster creation", func() {
		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createCluster(clusterName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("Check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return IsClusterActive(clusterName) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("Cluster %s is not active", clusterName))
		})
	})
})

func executeCloudCredentialsTemplate(data *cloudCredentialsData, buffer *bytes.Buffer) error {
	cloudCredentialsTemplate, err := template.New("cloudCredentials").Parse(cloudCredentialsRequestBodyTemplate)
	if err != nil {
		return fmt.Errorf("failed to create the cloud credentials template: %v", err)
	}
	return cloudCredentialsTemplate.Execute(buffer, *data)
}

func createCloudCredential(credentialName string) (string, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cloudcredentials")
	privateKeyContents, err := getFileContents(privateKeyPath)
	if err != nil {
		t.Logs.Infof("error reading private key file: %v", err)
		return "", err
	}
	credentialsData := cloudCredentialsData{
		CredentialName:     replaceWhitespaceToLiteral(credentialName),
		Fingerprint:        replaceWhitespaceToLiteral(fingerprint),
		PrivateKeyContents: replaceWhitespaceToLiteral(privateKeyContents),
		TenancyID:          replaceWhitespaceToLiteral(tenancyID),
		UserID:             replaceWhitespaceToLiteral(userID),
		Region:             replaceWhitespaceToLiteral(region),
		Passphrase:         "",
	}
	buf := &bytes.Buffer{}
	err = executeCloudCredentialsTemplate(&credentialsData, buf)
	if err != nil {
		return "", fmt.Errorf("failed to parse the cloud credentials template: " + err.Error())
	}

	request, err := retryablehttp.NewRequest(http.MethodPost, requestURL, buf)
	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		t.Logs.Errorf(err.Error())
		return "", err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	jsonBody, err := gabs.ParseJSON(body)
	if err != nil {
		return "", err
	}
	credID := fmt.Sprint(jsonBody.Path("id").Data())
	return credID, nil
}

func executeCreateClusterTemplate(data *capiClusterData, buffer *bytes.Buffer) error {
	createClusterTemplate, err := template.New("cloudCredentials").Parse(createClusterPayloadTemplate)
	if err != nil {
		return fmt.Errorf("failed to create the create cluster template: %v", err)
	}
	return createClusterTemplate.Execute(buffer, *data)
}

// Creates an OCNE cluster through CAPI
func createCluster(clusterName string) error {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster")
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath)
	if err != nil {
		t.Logs.Infof("error reading node public key file: %v", err)
		return err
	}
	capiClusterData := capiClusterData{
		ClusterName:           replaceWhitespaceToLiteral(clusterName),
		Region:                replaceWhitespaceToLiteral(region),
		VcnID:                 replaceWhitespaceToLiteral(vcnID),
		NodePublicKeyContents: replaceWhitespaceToLiteral(nodePublicKeyContents),
		CompartmentID:         replaceWhitespaceToLiteral(compartmentID),
		WorkerNodeSubnet:      replaceWhitespaceToLiteral(workerNodeSubnet),
		ControlPlaneSubnet:    replaceWhitespaceToLiteral(controlPlaneSubnet),
		LoadBalancerSubnet:    replaceWhitespaceToLiteral(loadBalancerSubnet),
		CloudCredentialID:     replaceWhitespaceToLiteral(cloudCredentialID),
		PodCIDR:               replaceWhitespaceToLiteral(podCidr),
	}
	buf := &bytes.Buffer{}
	err = executeCreateClusterTemplate(&capiClusterData, buf)
	if err != nil {
		return fmt.Errorf("failed to parse the cloud credentials template: " + err.Error())
	}
	request, err := retryablehttp.NewRequest(http.MethodPost, requestURL, buf)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	response, err := httpClient.Do(request)
	if err != nil {
		t.Logs.Errorf("error create cluster POST response: %v", response)
		return err
	}
	t.Logs.Infof("Create Cluster POST response: %v", response)
	defer response.Body.Close()
	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		t.Logs.Errorf(err.Error())
		return err
	}
	return err
}

// Returns true if the cluster currently exists and is Active
func IsClusterActive(clusterName string) (bool, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return false, err
	}
	t.Logs.Infof("jsonBody: %v" + jsonBody.String())
	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	t.Logs.Infof("State: " + state)
	return state == "active", nil
}

// Gets a specified cluster by using the Rancher REST API
func getCluster(clusterName string) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?name="+clusterName)
	request, err := retryablehttp.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %v", adminToken))
	request.Header.Add("Content-Type", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		t.Logs.Errorf(err.Error())
		return nil, err
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	jsonBody, err := gabs.ParseJSON(body)
	if err != nil {
		return nil, err
	}
	return jsonBody, nil
}

func replaceWhitespaceToLiteral(s string) string {
	modified := strings.ReplaceAll(s, "\n", `\n`)
	modified = strings.ReplaceAll(modified, "\t", `\t`)
	modified = strings.ReplaceAll(modified, "\v", `\v`)
	modified = strings.ReplaceAll(modified, "\r", `\r`)
	modified = strings.ReplaceAll(modified, "\f", `\f`)
	return modified
}

func getFileContents(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		t.Logs.Errorf("failed reading file contents: %v", err.Error())
		return "", err
	}
	return string(data), nil
}

func setupRequest(rancherBaseURL, urlPath string) (string, string) {
	adminToken := helpers.GetRancherLoginToken(t.Logs)
	t.Logs.Infof("adminToken: %s", adminToken)
	requestURL := fmt.Sprintf("%s/%s", rancherBaseURL, urlPath)
	t.Logs.Infof("createCluster requestURL: %s", requestURL)
	return requestURL, adminToken
}
