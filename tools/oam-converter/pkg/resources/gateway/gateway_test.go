// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gateway

import (
	"crypto/rand"
	"fmt"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	istio "istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/big"
	"testing"
)

func TestBuildCertificateSecretName(t *testing.T) {
	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-trait",
		},
	}
	appNamespace := getRandomString()

	expectedSecretName := fmt.Sprintf("%s-%s-cert-secret", appNamespace, trait.Name)

	result := buildCertificateSecretName(trait, appNamespace)

	assert.Equal(t, expectedSecretName, result, "Incorrect certificate secret name")
}

// Helper function to generate a random string
func getRandomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	max := big.NewInt(int64(len(charset)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {

			return ""
		}
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func TestBuildCertificateName(t *testing.T) {

	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-trait",
		},
	}
	appNamespace := getRandomString()

	expectedSecretName := fmt.Sprintf("%s-%s-cert", appNamespace, trait.Name)

	result := buildCertificateName(trait, appNamespace)

	assert.Equal(t, expectedSecretName, result, "Incorrect certificate name")
}

func TestBuildLegacyCertificateName(t *testing.T) {

	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-trait",
		},
	}
	appNamespace := getRandomString()

	expected := fmt.Sprintf("%s-%s-cert", appNamespace, trait.Name)

	appName := "my-app"
	result := buildLegacyCertificateName(trait, appNamespace, appName)

	assert.Equal(t, expected, result, "Incorrect legacy certificate name")
}

func TestBuildLegacyCertificateSecretName(t *testing.T) {
	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-trait",
		},
	}

	appName := "my-app"

	appNamespace := getRandomString()

	expected := fmt.Sprintf("%s-%s-cert-secret", appNamespace, trait.Name)

	result := buildLegacyCertificateSecretName(trait, appNamespace, appName)

	assert.Equal(t, expected, result, "Incorrect legacy certificate secret name")
}

func TestValidateConfiguredSecret(t *testing.T) {
	trait := &vzapi.IngressTrait{
		Spec: vzapi.IngressTraitSpec{
			TLS: vzapi.IngressSecurity{
				SecretName: "my-secret",
			},
		},
	}

	expected := "my-secret"
	result := validateConfiguredSecret(trait)

	assert.Equal(t, expected, result, "Incorrect configured secret")
}

func TestBuildGatewayName(t *testing.T) {
	appNamespace := "my-namespace"

	expected := "my-namespace-my-namespace-gw"
	result, err := BuildGatewayName(appNamespace)

	assert.NoError(t, err, "Error was not expected")
	assert.Equal(t, expected, result, "Incorrect gateway name")
}

func TestFormatGatewayServerPortName(t *testing.T) {
	traitName := "my-trait"

	expected := "https-my-trait"
	result := formatGatewayServerPortName(traitName)

	assert.Equal(t, expected, result, "Incorrect gateway server port name")
}
func TestCreateGatewayCertificate(t *testing.T) {
	trait := &vzapi.IngressTrait{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-trait",
			Namespace: "my-namespace",
		},
	}

	hostsForTrait := []string{"example.com"}

	appNamespace := getRandomString()

	expectedSecretName := fmt.Sprintf("%s-%s-cert-secret", appNamespace, trait.Name)

	secretName := createGatewayCertificate(trait, hostsForTrait, appNamespace)

	// Check the certificate name and secret name returned by the function
	assert.Equal(t, expectedSecretName, secretName, "Incorrect secret name")

}

func TestUpdateGatewayServersList(t *testing.T) {
	// Create a sample server
	server1 := &istio.Server{
		Name: "server1",
		Port: &istio.Port{Name: "https"},
	}

	// Create another sample server with a different name
	server2 := &istio.Server{
		Name: "server1",
		Port: &istio.Port{Name: "http"},
	}

	// Create an empty server list
	var servers []*istio.Server

	// Call the function with the empty server list and the first server
	servers = updateGatewayServersList(servers, server1)

	// The server list should now have only one server
	assert.Len(t, servers, 1, "Unexpected number of servers")
	assert.Equal(t, server1, servers[0], "Unexpected server in the list")

	// Call the function again with the same server list and a different server
	servers = updateGatewayServersList(servers, server2)

	// The server list should still have only one server with the updated details
	assert.Len(t, servers, 1, "Unexpected number of servers")
	assert.Equal(t, server2, servers[0], "Unexpected server in the list")

	// Call the function with an empty server list and a new server
	servers = updateGatewayServersList(nil, server1)

	// The server list should now have the new server
	assert.Len(t, servers, 1, "Unexpected number of servers")
	assert.Equal(t, server1, servers[0], "Unexpected server in the list")

	// Call the function with a server list containing an unnamed server
	server3 := &istio.Server{
		Port: &istio.Port{Name: "https"},
	}
	servers = updateGatewayServersList([]*istio.Server{server3}, server1)

	// The server list should now have the updated server with a name
	assert.Len(t, servers, 1, "Unexpected number of servers")
	assert.Equal(t, server1, servers[0], "Unexpected server in the list")
}

func TestCreateGatewayResource(t *testing.T) {
	conversionComponent := &types.ConversionComponents{
		AppName:      "hello-helidon",
		AppNamespace: "my-namespace",
		IngressTrait: &vzapi.IngressTrait{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-namespace",
			},
		},
	}
	// Create a sample list of conversion components
	conversionComponents := []*types.ConversionComponents{

		conversionComponent,
	}

	// Call the function with the sample conversion components
	gateway, _, err := CreateGatewayResource(conversionComponents)

	// Check if the function returns an error (if any)
	assert.NoError(t, err, "Unexpected error returned")

	// Check if the gateway object is not nil
	assert.NotNil(t, gateway, "Gateway object is nil")

}

func TestCreateListGateway(t *testing.T) {
	// Create a sample Gateway object
	gateway := &vsapi.Gateway{}

	// Call the function with the sample Gateway
	gatewayData, err := CreateListGateway(gateway)

	// Check if the function returns an error (if any)
	assert.NoError(t, err, "Unexpected error returned")

	// Check if the gatewayData map is not nil
	assert.NotNil(t, gatewayData, "gatewayData map is nil")

	// Check if the "apiVersion" field is set to "v1"
	assert.Equal(t, "v1", gatewayData["apiVersion"], "Unexpected apiVersion")

	// Check if the "items" field is an array containing the sample Gateway
	gatewayItems, ok := gatewayData["items"].([]*vsapi.Gateway)
	assert.True(t, ok, "items is not an array of Gateway")
	assert.Len(t, gatewayItems, 1, "Unexpected number of items in the array")
	assert.Equal(t, gateway, gatewayItems[0], "Unexpected Gateway in the items array")

	// Check if the "kind" field is set to "List"
	assert.Equal(t, "List", gatewayData["kind"], "Unexpected kind")

}
