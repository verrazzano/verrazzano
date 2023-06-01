// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"go.uber.org/zap"

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
	clusterName                  = "strudel2"
	createClusterPayloadTemplate = `{
		"dockerRootDir": "/var/lib/docker",
		"enableClusterAlerting": false,
		"enableClusterMonitoring": false,
		"enableNetworkPolicy": false,
		"windowsPreferedCluster": false,
		"type": "cluster",
		"name": "{{.ClusterName}}",
		"ociocneEngineConfig": {
			"calicoImagePath": "olcne",
			"calicoImageRegistry": "container-registry.oracle.com",
			"ccmImage": "ghcr.io/oracle/cloud-provider-oci:v1.24.0",
			"cloudCredentialId": "{{.CloudCredentialID}}",
			"clusterCidr": "10.96.0.0/16",
			"compartmentId": "{{.CompartmentID}}",
			"controlPlaneMemoryGbs": 16,
			"controlPlaneOcpus": 2,
			"controlPlaneRegistry": "container-registry.oracle.com/olcne",
			"controlPlaneShape": "VM.Standard.E4.Flex",
			"controlPlaneSubnet": "{{.ControlPlaneSubnet}}",
			"controlPlaneVolumeGbs": 100,
			"csiRegistry": "k8s.gcr.io/sig-storage",
			"displayName": "{{.ClusterName}}",
			"driverName": "ociocneengine",
			"imageDisplayName": "Oracle-Linux-8.7-2023.04.25-0",
			"imageId": "",
			"installCalico": true,
			"installCcm": true,
			"installCsi": true,
			"installVerrazzano": false,
			"kubernetesVersion": "v1.24.8",
			"loadBalancerSubnet": "{{.LoadBalancerSubnet}}",
			"name": "",
			"nodePublicKeyContents": "{{.NodePublicKeyContents}}",
			"numControlPlaneNodes": 1,
			"ociCsiImage": "ghcr.io/oracle/cloud-provider-oci:v1.24.0",
			"podCidr": "192.168.0.0/16",
			"proxyEndpoint": "",
			"region": "{{.Region}}",
			"skipOcneInstall": false,
			"useNodePvEncryption": true,
			"vcnId": "{{.VcnID}}",
			"verrazzanoImage": "ghcr.io/verrazzano/verrazzano-platform-operator:v1.5.2-20230315235330-0326ee67",
			"verrazzanoResource": "apiVersion: install.verrazzano.io/v1beta1\nkind: Verrazzano\nmetadata:\n  name: managed\n  namespace: default\nspec:\n  profile: managed-cluster",
			"workerNodeSubnet": "{{.WorkerNodeSubnet}}",
			"type": "ociocneEngineConfig",
			"clusterName": "",
			"nodeShape": "VM.Standard.E4.Flex",
			"numWorkerNodes": 1,
			"nodePools": [],
			"applyYamls": []
		},
		"cloudCredentialId": "{{.CloudCredentialID}}",
		"labels": {}
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
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	Skip("Temporarily skipping the test suite since the test suite is not passing/unstable")

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

var afterSuite = t.AfterSuiteFunc(func() {
	// TODO: delete cloud credential(s)
})
var _ = AfterSuite(afterSuite)

var _ = t.Describe("OCNE Cluster Driver", Label("f:rancher-capi:ocne-cluster-driver"), Serial, func() {
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

	t.Context("OCNE cluster delete", func() {
		t.It("delete OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return deleteCluster(clusterName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("Check OCNE cluster is deleted", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return IsClusterDeleted(clusterName) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("Cluster %s is not active", clusterName))
		})
	})
})

func deleteCluster(clusterName string) error {
	clusterID, err := getClusterIDFromName(clusterName)
	if err != nil {
		t.Logs.Infof("Could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return err
	}
	t.Logs.Infof("clusterID for deletion: %s", clusterID)

	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s/%s", "v1/provisioning.cattle.io.clusters/fleet-default", clusterID))

	_, err = helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while deleting cluster: %v", err)
		return err
	}
	return nil
}

func IsClusterDeleted(clusterName string) (bool, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return false, err
	}
	jsonData := fmt.Sprint(jsonBody.Path("data").Data())
	return jsonData == "[]", nil
}

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
		return "", fmt.Errorf("failed to parse the cloud credentials template: %s", err.Error())
	}

	jsonBody, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, buf.Bytes(), t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data: %v", zap.Error(err))
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
	}
	buf := &bytes.Buffer{}
	err = executeCreateClusterTemplate(&capiClusterData, buf)
	if err != nil {
		return fmt.Errorf("failed to parse the cloud credentials template: %s", err.Error())
	}
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, buf.Bytes(), t.Logs)
	if res != nil {
		//t.Logs.Infof("create cluster response body: %s", res.String())
	}
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data: %v", zap.Error(err))
		return err
	}
	return nil
}

// Returns true if the cluster currently exists and is Active
func IsClusterActive(clusterName string) (bool, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return false, err
	}
	//t.Logs.Infof("jsonBody: %s", jsonBody.String())
	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	t.Logs.Infof("State: %s", state)
	return state == "active", nil
}

func getClusterIDFromName(clusterName string) (string, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(jsonBody.Path("data.0.id").Data()), nil
}

// Gets a specified cluster by using the Rancher REST API
func getCluster(clusterName string) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?name="+clusterName)
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
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
		t.Logs.Errorf("failed reading file contents: %v", err)
		return "", err
	}
	return string(data), nil
}

func setupRequest(rancherBaseURL, urlPath string) (string, string) {
	adminToken := helpers.GetRancherLoginToken(t.Logs)
	t.Logs.Infof("adminToken: %s", adminToken)
	requestURL := fmt.Sprintf("%s/%s", rancherBaseURL, urlPath)
	t.Logs.Infof("requestURL: %s", requestURL)
	return requestURL, adminToken
}
