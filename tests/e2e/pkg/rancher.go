// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"go.uber.org/zap"
	"io"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"net/http"
	"strconv"
	"strings"
)

type Payload struct {
	ClusterID string `json:"clusterID"`
	TTL       int    `json:"ttl"`
}

type TokenPostResponse struct {
	Token   string `json:"token"`
	Created string `json:"created"`
}

//
//type Token struct {
//	AuthProvider string      `json:"authProvider"`
//	BaseType     string      `json:"baseType"`
//	ClusterID    string      `json:"clusterId"`
//	Created      string      `json:"created"`
//	CreatedTS    int64       `json:"createdTS"`
//	CreatorID    interface{} `json:"creatorId"`
//	Current      bool        `json:"current"`
//	Description  string      `json:"description"`
//	Enabled      bool        `json:"enabled"`
//	Expired      bool        `json:"expired"`
//	ExpiresAt    string      `json:"expiresAt"`
//	ID           string      `json:"id"`
//	IsDerived    bool        `json:"isDerived"`
//	Labels       struct {
//		AuthnManagementCattleIoTokenUserID string `json:"authn.management.cattle.io/token-userId"`
//		CattleIoCreator                    string `json:"cattle.io/creator"`
//	} `json:"labels"`
//	LastUpdateTime string `json:"lastUpdateTime"`
//	Links          struct {
//		Remove string `json:"remove"`
//		Self   string `json:"self"`
//		Update string `json:"update"`
//	} `json:"links"`
//	Name          string `json:"name"`
//	TTL           int    `json:"ttl"`
//	Type          string `json:"type"`
//	UserID        string `json:"userId"`
//	UserPrincipal string `json:"userPrincipal"`
//	UUID          string `json:"uuid"`
//}

type ListOfTokenOutputFromRancher struct {
	Type  string `json:"type"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	CreateTypes struct {
		Token string `json:"token"`
	} `json:"createTypes"`
	Actions struct {
	} `json:"actions"`
	Pagination struct {
		Limit int `json:"limit"`
	} `json:"pagination"`
	Sort struct {
		Order   string `json:"order"`
		Reverse string `json:"reverse"`
		Links   struct {
			AuthProvider   string `json:"authProvider"`
			Description    string `json:"description"`
			ExpiresAt      string `json:"expiresAt"`
			LastUpdateTime string `json:"lastUpdateTime"`
			Token          string `json:"token"`
			UUID           string `json:"uuid"`
		} `json:"links"`
	} `json:"sort"`
	Filters struct {
		AuthProvider interface{} `json:"authProvider"`
		ClusterID    interface{} `json:"clusterId"`
		Created      interface{} `json:"created"`
		CreatorID    interface{} `json:"creatorId"`
		Current      []struct {
			Modifier string `json:"modifier"`
			Value    string `json:"value"`
		} `json:"current"`
		Description    interface{} `json:"description"`
		Enabled        interface{} `json:"enabled"`
		Expired        interface{} `json:"expired"`
		ExpiresAt      interface{} `json:"expiresAt"`
		IsDerived      interface{} `json:"isDerived"`
		LastUpdateTime interface{} `json:"lastUpdateTime"`
		Name           interface{} `json:"name"`
		Removed        interface{} `json:"removed"`
		Token          interface{} `json:"token"`
		TTL            interface{} `json:"ttl"`
		UserID         interface{} `json:"userId"`
		UserPrincipal  interface{} `json:"userPrincipal"`
		UUID           interface{} `json:"uuid"`
	} `json:"filters"`
	ResourceType string `json:"resourceType"`
	Data         []struct {
		AuthProvider string      `json:"authProvider"`
		BaseType     string      `json:"baseType"`
		ClusterID    string      `json:"clusterId"`
		Created      string      `json:"created"`
		CreatedTS    int64       `json:"createdTS"`
		CreatorID    interface{} `json:"creatorId"`
		Current      bool        `json:"current"`
		Description  string      `json:"description"`
		Enabled      bool        `json:"enabled"`
		Expired      bool        `json:"expired"`
		ExpiresAt    string      `json:"expiresAt"`
		ID           string      `json:"id"`
		IsDerived    bool        `json:"isDerived"`
		Labels       struct {
			AuthnManagementCattleIoTokenUserID string `json:"authn.management.cattle.io/token-userId"`
			CattleIoCreator                    string `json:"cattle.io/creator"`
		} `json:"labels"`
		LastUpdateTime string `json:"lastUpdateTime"`
		Links          struct {
			Remove string `json:"remove"`
			Self   string `json:"self"`
			Update string `json:"update"`
		} `json:"links"`
		Name          string `json:"name"`
		TTL           int    `json:"ttl"`
		Type          string `json:"type"`
		UserID        string `json:"userId"`
		UserPrincipal string `json:"userPrincipal"`
		UUID          string `json:"uuid"`
	} `json:"data"`
}

func EventuallyGetURLForIngress(log *zap.SugaredLogger, api *APIEndpoint, namespace string, name string, scheme string) string {
	ingressHost := EventuallyGetIngressHost(log, api, namespace, name)
	gomega.Expect(ingressHost).ToNot(gomega.BeEmpty())
	return fmt.Sprintf("%s://%s", scheme, ingressHost)
}

func EventuallyGetIngressHost(log *zap.SugaredLogger, api *APIEndpoint, namespace string, name string) string {
	var ingressHost string
	gomega.Eventually(func() error {
		ingress, err := api.GetIngress(namespace, name)
		if err != nil {
			return err
		}
		if len(ingress.Spec.Rules) == 0 {
			return fmt.Errorf("no rules found in ingress %s/%s", namespace, name)
		}
		ingressHost = ingress.Spec.Rules[0].Host
		log.Info(fmt.Sprintf("Found ingress host: %s", ingressHost))
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil())
	return ingressHost
}

func GetURLForIngress(log *zap.SugaredLogger, api *APIEndpoint, namespace string, name string, scheme string) (string, error) {
	ingress, err := api.GetIngress(namespace, name)
	if err != nil {
		return "", err
	}
	ingressURL := fmt.Sprintf("%s://%s", scheme, ingress.Spec.Rules[0].Host)
	log.Info(fmt.Sprintf("Found ingress URL: %s", ingressURL))
	return ingressURL, err
}

func eventuallyGetRancherAdminPassword(log *zap.SugaredLogger) (string, error) {
	var err error
	var secret *corev1.Secret
	gomega.Eventually(func() error {
		secret, err = GetSecret("cattle-system", "rancher-admin-secret")
		if err != nil {
			log.Error(fmt.Sprintf("Error getting rancher-admin-secret, retrying: %v", err))
		}
		return err
	}, waitTimeout, pollingInterval).Should(gomega.BeNil())

	if secret == nil {
		return "", fmt.Errorf("Unable to get rancher admin secret")
	}

	var rancherAdminPassword []byte
	var ok bool
	if rancherAdminPassword, ok = secret.Data["password"]; !ok {
		return "", fmt.Errorf("Error getting rancher admin credentials")
	}

	return string(rancherAdminPassword), nil
}

func GetRancherAdminToken(log *zap.SugaredLogger, httpClient *retryablehttp.Client, rancherURL string) string {
	rancherAdminPassword, err := eventuallyGetRancherAdminPassword(log)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting rancher admin password: %v", err))
		return ""
	}

	token, err := getRancherUserToken(log, httpClient, rancherURL, "admin", string(rancherAdminPassword))
	if err != nil {
		log.Error(fmt.Sprintf("Error getting user token from rancher: %v", err))
		return ""
	}

	return token
}

func getRancherUserToken(log *zap.SugaredLogger, httpClient *retryablehttp.Client, rancherURL string, username string, password string) (string, error) {
	rancherLoginURL := fmt.Sprintf("%s/%s", rancherURL, "v3-public/localProviders/local?action=login")
	payload := `{"Username": "` + username + `", "Password": "` + password + `"}`
	response, err := httpClient.Post(rancherLoginURL, "application/json", strings.NewReader(payload))
	if err != nil {
		log.Error(fmt.Sprintf("Error getting rancher admin token: %v", err))
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		log.Errorf("Invalid response code when fetching Rancher token: %v", err)
		return "", err
	}

	defer response.Body.Close()

	// extract the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read Rancher token response: %v", err)
		return "", err
	}

	token, err := httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "token", "unable to find token in Rancher response")
	if err != nil {
		log.Errorf("Failed to extract token from Rancher response: %v", err)
		return "", err
	}

	return token, nil
}

func AddAccessTokenToRancherForLoggedInUser(httpClient *retryablehttp.Client, kubeconfigPath string, clusterID string, ttl string, userAccessToken string, log zap.SugaredLogger) (string, error) {
	api, err := GetAPIEndpoint(kubeconfigPath)
	if err != nil {
		log.Errorf("API Endpoint not successfully received based on KubeConfig Path")
		return "", err
	}
	rancherURL, err := GetURLForIngress(&log, api, "cattle-system", "rancher", "https")
	if err != nil {
		log.Errorf("URL For Rancher not successfully found")
		return "", err
	}
	val, _ := strconv.Atoi(ttl)
	payload := &Payload{
		ClusterID: clusterID,
		TTL:       val * 60000,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	reqURL := rancherURL + "/v3/tokens"

	req, err := retryablehttp.NewRequest("POST", reqURL, data)
	if err != nil {
		return "", err
	}
	req.Header = map[string][]string{"Authorization": {"Bearer " + userAccessToken}, "Content-Type": {"application/json"}}

	response, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return "", err
	}

	var tokenPostResponse TokenPostResponse
	err = json.Unmarshal([]byte(responseBody), &tokenPostResponse)
	if err != nil {
		return "", err
	}

	return tokenPostResponse.Created, nil
}

func GetAndDeleteTokenNamesForLoggedInUserBasedOnClusterID(httpClient *retryablehttp.Client, kubeconfigPath string, clusterID string, userAccessToken string, log zap.SugaredLogger) ([]string, error) {
	api, err := GetAPIEndpoint(kubeconfigPath)
	if err != nil {
		log.Errorf("API Endpoint not successfully received based on KubeConfig Path")
		return nil, err
	}
	rancherURL, err := GetURLForIngress(&log, api, "cattle-system", "rancher", "https")
	if err != nil {
		log.Errorf("URL For Rancher not successfully found")
		return nil, err
	}
	reqURL := rancherURL + "/v3/tokens"

	getReq, err := retryablehttp.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	getReq.Header = map[string][]string{"Authorization": {"Bearer " + userAccessToken}, "Content-Type": {"application/json"}, "Accept": {"application/json"}}
	response, err := httpClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	fmt.Println(response.Body)
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return nil, err
	}
	var listOfTokenOutputFromRancher = ListOfTokenOutputFromRancher{}
	err = json.Unmarshal(responseBody, &listOfTokenOutputFromRancher)
	if err != nil {
		return nil, err
	}
	listOfTokens := listOfTokenOutputFromRancher.Data
	var listOfTokensToReturn = make([]string, 0)

	for _, token := range listOfTokens {
		//Check if token is valid, if it is token that we want than append it
		if token.ClusterID == clusterID {
			listOfTokensToReturn = append(listOfTokensToReturn, token.Name)
		}

	}
	for _, tokenName := range listOfTokensToReturn {
		//Check that it is not the same name as the user access token
		if tokenName == userAccessToken {
			continue
		}
		deleteSingleTokenURL := reqURL + "/" + tokenName
		deleteReq, err := retryablehttp.NewRequest("DELETE", deleteSingleTokenURL, nil)
		if err != nil {
			return nil, err
		}
		deleteReq.Header = map[string][]string{"Authorization": {"Bearer " + userAccessToken}, "Accept": {"application/json"}}
		response, err := httpClient.Do(deleteReq)
		if err != nil {
			return nil, err
		}
		err = httputil.ValidateResponseCode(response, http.StatusNoContent)
		if err != nil {
			return nil, err
		}
	}

	return listOfTokensToReturn, nil

}

// VerifyRancherAccess verifies that Rancher is accessible.
func VerifyRancherAccess(log *zap.SugaredLogger) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
		return err
	}

	api := EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherURL := EventuallyGetURLForIngress(log, api, "cattle-system", "rancher", "https")
	httpClient := EventuallyVerrazzanoRetryableHTTPClient()
	var httpResponse *HTTPResponse

	gomega.Eventually(func() (*HTTPResponse, error) {
		httpResponse, err = GetWebPageWithClient(httpClient, rancherURL, "")
		return httpResponse, err
	}, waitTimeout, pollingInterval).Should(HasStatus(http.StatusOK))

	gomega.Expect(CheckNoServerHeader(httpResponse)).To(gomega.BeTrue(), "Found unexpected server header in response")
	return nil
}

// VerifyRancherKeycloakAuthConfig verifies that Rancher/Keycloak AuthConfig is correctly populated
func VerifyRancherKeycloakAuthConfig(log *zap.SugaredLogger) error {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		log.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
		return err
	}

	log.Info("Verify Keycloak AuthConfig")

	gomega.Eventually(func() (bool, error) {
		api, err := GetAPIEndpoint(kubeconfigPath)
		if err != nil {
			log.Error(fmt.Sprintf("Error getting API endpoint: %v", err))
			return false, err
		}
		keycloakURL, err := GetURLForIngress(log, api, "keycloak", "keycloak", "https")
		if err != nil {
			log.Error(fmt.Sprintf("Error getting API endpoint: %v", err))
			return false, err
		}
		rancherURL, err := GetURLForIngress(log, api, "cattle-system", "rancher", "https")
		if err != nil {
			return false, err
		}
		k8sClient, err := GetDynamicClientInCluster(kubeconfigPath)
		if err != nil {
			log.Error(fmt.Sprintf("Error getting dynamic client: %v", err))
			return false, err
		}

		authConfigData, err := k8sClient.Resource(GvkToGvr(common.GVKAuthConfig)).Get(context.Background(), common.AuthConfigKeycloak, v1.GetOptions{})
		if err != nil {
			log.Error(fmt.Sprintf("error getting keycloak oidc authConfig: %v", err))
			return false, err
		}

		authConfigAttributes := authConfigData.UnstructuredContent()
		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeAccessMode, authConfigAttributes[rancher.AuthConfigKeycloakAttributeAccessMode].(string), rancher.AuthConfigKeycloakAccessMode); err != nil {
			log.Error(err)
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeClientID, authConfigAttributes[rancher.AuthConfigKeycloakAttributeClientID].(string), rancher.AuthConfigKeycloakClientIDRancher); err != nil {
			log.Error(err)
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeGroupSearchEnabled, authConfigAttributes[rancher.AuthConfigKeycloakAttributeGroupSearchEnabled].(bool), true); err != nil {
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeAuthEndpoint, authConfigAttributes[rancher.AuthConfigKeycloakAttributeAuthEndpoint].(string), keycloakURL+rancher.AuthConfigKeycloakURLPathAuthEndPoint); err != nil {
			log.Error(err)
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeRancherURL, authConfigAttributes[rancher.AuthConfigKeycloakAttributeRancherURL].(string), rancherURL+rancher.AuthConfigKeycloakURLPathVerifyAuth); err != nil {
			log.Error(err)
			return false, err
		}

		authConfigClientSecret := authConfigAttributes[common.AuthConfigKeycloakAttributeClientSecret].(string)
		if authConfigClientSecret == "" {
			err = fmt.Errorf("keycloak auth config attribute %s not correctly configured, value is empty", common.AuthConfigKeycloakAttributeClientSecret)
			log.Error(err)
			return false, err
		}

		return true, nil
	}, waitTimeout, pollingInterval).Should(gomega.Equal(true), "keycloak oidc authconfig not configured correctly")
	return nil
}

// GvkToGvr converts a GroupVersionKind to corresponding GroupVersionResource
func GvkToGvr(gvk schema.GroupVersionKind) schema.GroupVersionResource {
	resource := strings.ToLower(gvk.Kind)
	if strings.HasSuffix(resource, "s") {
		resource = resource + "es"
	} else {
		resource = resource + "s"
	}

	return schema.GroupVersionResource{Group: gvk.Group,
		Version:  gvk.Version,
		Resource: resource,
	}
}

func verifyAuthConfigAttribute(name string, actual interface{}, expected interface{}) error {
	if expected != actual {
		return fmt.Errorf("keycloak auth config attribute %s not correctly configured, expected %v, actual %v", name, expected, actual)
	}
	return nil
}

func EventuallyGetRancherHost(log *zap.SugaredLogger, api *APIEndpoint) (string, error) {
	rancherHost := EventuallyGetIngressHost(log, api, rancher.ComponentNamespace, common.RancherName)
	if rancherHost == "" {
		return "", fmt.Errorf("got empty Rancher ingress host")
	}
	return rancherHost, nil
}

func CreateNewRancherConfig(log *zap.SugaredLogger, kubeconfigPath string) (*rancherutil.RancherConfig, error) {
	rancherAdminPassword, err := eventuallyGetRancherAdminPassword(log)
	if err != nil {
		return nil, err
	}
	return CreateNewRancherConfigForUser(log, kubeconfigPath, "admin", rancherAdminPassword)
}

func CreateNewRancherConfigForUser(log *zap.SugaredLogger, kubeconfigPath string, username string, password string) (*rancherutil.RancherConfig, error) {
	apiEndpoint := EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherHost, err := EventuallyGetRancherHost(log, apiEndpoint)
	if err != nil {
		return nil, err
	}
	rancherURL := fmt.Sprintf("https://%s", rancherHost)
	caCert, err := GetCACertFromSecret(common.RancherIngressCAName, constants.RancherSystemNamespace, "ca.crt", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get caCert: %v", err)
	}

	// the tls-ca-additional secret is optional
	additionalCA, _ := GetCACertFromSecret(constants.AdditionalTLS, constants.RancherSystemNamespace, constants.AdditionalTLSCAKey, kubeconfigPath)

	httpClient, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	token, err := getRancherUserToken(log, httpClient, rancherURL, username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to get user token from Rancher: %v", err)
	}

	rc := rancherutil.RancherConfig{
		// populate Rancher config from the functions available in this file,adding as necessary
		BaseURL:                  rancherURL,
		Host:                     rancherHost,
		APIAccessToken:           token,
		CertificateAuthorityData: caCert,
		AdditionalCA:             additionalCA,
	}
	return &rc, nil
}

func GetClusterKubeconfig(log *zap.SugaredLogger, httpClient *retryablehttp.Client, rc *rancherutil.RancherConfig, clusterID string) (string, error) {
	reqURL := rc.BaseURL + "/v3/clusters/" + clusterID + "?action=generateKubeconfig"
	req, err := retryablehttp.NewRequest("POST", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+rc.APIAccessToken)

	response, err := httpClient.Do(req)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting managed cluster kubeconfig: %v", err))
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		log.Errorf("Invalid response code when fetching cluster kubeconfig: %v", err)
		return "", err
	}

	defer response.Body.Close()

	// extract the response body
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read Rancher kubeconfig response: %v", err)
		return "", err
	}

	return httputil.ExtractFieldFromResponseBodyOrReturnError(string(responseBody), "config", "")
}
