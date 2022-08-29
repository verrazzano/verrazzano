// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"

	"io/ioutil"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KeycloakRESTClient struct {
	kubeConfigPath      string
	keycloakIngressHost string
	adminAccessToken    string
	httpClient          *retryablehttp.Client
}

const (
	keycloakNamespace               = "keycloak"
	keycloadIngressName             = "keycloak"
	keycloakAdminUserPasswordSecret = "keycloak-http" //nolint:gosec //#gosec G101
	keycloakAdminUserRealm          = "master"
	keycloakAdminUserName           = "keycloakadmin"

	keycloakAPIClientID   = "verrazzano-pg"
	keycloakAdminClientID = "admin-cli"

	TestKeycloakMasterUserIDKey     = "TEST_KEYCLOAK_MASTER_USERID"
	TestKeycloakVerrazzanoUserIDKey = "TEST_KEYCLOAK_VZ_USERID"

	TestKeycloakNamespace = "keycloak-test-ns"
	TestKeycloakConfigMap = "keycloak-test-cm"
)

// NewKeycloakRESTClient creates a new Keycloak REST client.
func NewKeycloakAdminRESTClient() (*KeycloakRESTClient, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return nil, err
	}

	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	ingress, err := clientset.NetworkingV1().Ingresses(keycloakNamespace).Get(context.TODO(), keycloadIngressName, k8smeta.GetOptions{})
	if err != nil {
		return nil, err
	}
	httpClient, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	secret, err := GetSecret(keycloakNamespace, keycloakAdminUserPasswordSecret)
	if err != nil {
		return nil, err
	}
	keycloakAdminPassword := strings.TrimSpace(string(secret.Data["password"]))

	ingressHost := ingress.Spec.Rules[0].Host
	keycloakLoginURL := fmt.Sprintf("https://%s/auth/realms/%s/protocol/openid-connect/token", ingressHost, keycloakAdminUserRealm)
	body := fmt.Sprintf("username=%s&password=%s&grant_type=password&client_id=%s", keycloakAdminUserName, keycloakAdminPassword, keycloakAdminClientID)
	resp, err := PostWithHostHeader(keycloakLoginURL, "application/x-www-form-urlencoded", ingressHost, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to login as admin user")
	}
	token := JTq(string(resp.Body), "access_token").(string)
	if token == "" {
		return nil, fmt.Errorf("failed to obtain valid access token")
	}

	client := KeycloakRESTClient{
		kubeConfigPath:      kubeconfigPath,
		keycloakIngressHost: ingress.Spec.Rules[0].Host,
		adminAccessToken:    token,
		httpClient:          httpClient}
	return &client, nil
}

// GetRealm gets realm data from Keycloak.
func (c *KeycloakRESTClient) GetRealm(realm string) (map[string]interface{}, error) {
	requestURL := fmt.Sprintf("https://%s/auth/admin/realms/%s", c.keycloakIngressHost, realm)
	request, err := retryablehttp.NewRequest("GET", requestURL, nil)
	request.Host = c.keycloakIngressHost
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", c.adminAccessToken))
	request.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("invalid response")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("invalid response status: %d", response.StatusCode)
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	jsonMap := make(map[string]interface{})
	err = json.Unmarshal(responseBody, &jsonMap)
	if err != nil {
		return nil, err
	}
	return jsonMap, nil
}

// GetRealm gets a bearer token from a realm.
func (c *KeycloakRESTClient) GetToken(realm string, username string, password string, clientid string, log *zap.SugaredLogger) (string, error) {
	form := url.Values{}
	form.Add("username", username)
	form.Add("password", password)
	form.Add("grant_type", "password")
	form.Add("client_id", clientid)

	requestURL := fmt.Sprintf("https://%s/auth/realms/%s/protocol/openid-connect/token", c.keycloakIngressHost, realm)
	response, err := PostWithHostHeader(requestURL, "application/x-www-form-urlencoded", c.keycloakIngressHost, strings.NewReader(form.Encode()))
	log.Debugf("response: %s", response.Body)
	if response.StatusCode != 200 {
		return "", fmt.Errorf("invalid response status: %d", response.StatusCode)
	}
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to access token endpoint")
	}
	token := JTq(string(response.Body), "access_token").(string)
	if token == "" {
		return "", fmt.Errorf("failed to obtain valid access token")
	}

	return token, nil
}

// CreateUser creates a user in Keycloak
// curl -v http://localhost:8080/auth/admin/realms/apiv2/users -H "Content-Type: application/json" -H "Authorization: bearer $TOKEN"   --data '{"username":"someuser", "firstName":"xyz", "lastName":"xyz", "email":"demo2@gmail.com", "enabled":"true"}'
func (c *KeycloakRESTClient) CreateUser(userRealm string, userName string, firstName string, lastName string, password string) (string, error) {
	requestData := map[string]interface{}{
		"username":  userName,
		"firstName": firstName,
		"lastName":  lastName,
		"credentials": [...]map[string]interface{}{{
			"type":      "password",
			"value":     password,
			"temporary": false},
		},
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("marshal request failed: %v\n", err)
		return "", err
	}

	requestURL := fmt.Sprintf("https://%s/auth/admin/realms/%s/users", c.keycloakIngressHost, userRealm)
	request, err := retryablehttp.NewRequest("POST", requestURL, requestBody)
	if err != nil {
		return "", err
	}
	request.Host = c.keycloakIngressHost
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", c.adminAccessToken))
	request.Header.Add("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	if response == nil {
		return "", fmt.Errorf("invalid response")
	}
	defer response.Body.Close()
	location := response.Header.Get("Location")
	if response.StatusCode != 201 {
		return location, fmt.Errorf("invalid response status code: %d", response.StatusCode)
	}
	if location == "" {
		return location, fmt.Errorf("invalid response location")
	}
	return location, nil
}

// DeleteUser deletes a user from Keycloak
// DELETE /auth/admin/realms/<realm>/users/<userID>
func (c *KeycloakRESTClient) DeleteUser(userRealm string, userID string) error {
	requestURL := fmt.Sprintf("https://%s/auth/admin/realms/%s/users/%s", c.keycloakIngressHost, userRealm, userID)
	request, err := retryablehttp.NewRequest("DELETE", requestURL, nil)
	if err != nil {
		return err
	}
	request.Host = c.keycloakIngressHost
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", c.adminAccessToken))
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("invalid response")
	}
	defer response.Body.Close()
	// all responses in the 200 range are acceptable
	// in practice, this call returns 204 (No Content) for success
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("invalid response status: %d", response.StatusCode)
	}
	return nil
}

// VerifyUserExists verifies the user exists in Keycloak
// GET /auth/admin/realms/<realm>/users/<userID>
func (c *KeycloakRESTClient) VerifyUserExists(userRealm string, userID string) (bool, error) {
	requestURL := fmt.Sprintf("https://%s/auth/admin/realms/%s/users/%s", c.keycloakIngressHost, userRealm, userID)
	request, err := retryablehttp.NewRequest("GET", requestURL, nil)
	if err != nil {
		return false, err
	}
	request.Host = c.keycloakIngressHost
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", c.adminAccessToken))
	response, err := c.httpClient.Do(request)
	if err != nil {
		return false, err
	}
	if response == nil {
		return false, fmt.Errorf("invalid response")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return false, fmt.Errorf("invalid response status: %d", response.StatusCode)
	}
	return true, nil
}

// SetPassword sets a user's password in Keycloak
// PUT /auth/admin/realms/{realm}/users/{id}/reset-password
// { "type": "password", "temporary": false, "value": "..." }
func (c *KeycloakRESTClient) SetPassword(userRealm string, userID string, password string) error {
	requestData := map[string]interface{}{
		"type":      "password",
		"value":     password,
		"temporary": false}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return err
	}
	requestURL := fmt.Sprintf("https://%s/auth/admin/realms/%s/users/%s/reset-password", c.keycloakIngressHost, userRealm, userID)
	request, err := retryablehttp.NewRequest("PUT", requestURL, requestBody)
	if err != nil {
		fmt.Printf("create reset-password request failed=%v\n", err)
		return err
	}
	request.Host = c.keycloakIngressHost
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %v", c.adminAccessToken))
	request.Header.Add("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("invalid response")
	}
	defer response.Body.Close()
	if response.StatusCode != 204 {
		return fmt.Errorf("invalid response status: %d", response.StatusCode)
	}
	return nil
}
