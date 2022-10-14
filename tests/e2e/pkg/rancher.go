// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func EventuallyGetURLForIngress(log *zap.SugaredLogger, api *APIEndpoint, namespace string, name string, scheme string) string {
	var ingressURL string
	gomega.Eventually(func() error {
		ingress, err := api.GetIngress(namespace, name)
		if err != nil {
			return err
		}
		ingressURL = fmt.Sprintf("%s://%s", scheme, ingress.Spec.Rules[0].Host)
		log.Info(fmt.Sprintf("Found ingress URL: %s", ingressURL))
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil())
	gomega.Expect(ingressURL).ToNot(gomega.BeEmpty())
	return ingressURL
}

func GetRancherAdminToken(log *zap.SugaredLogger, httpClient *retryablehttp.Client, rancherURL string) string {
	var err error
	secret, err := GetSecret("cattle-system", "rancher-admin-secret")
	if err != nil {
		log.Error(fmt.Sprintf("Error getting rancher-admin-secret: %v", err))
		return ""
	}

	var rancherAdminPassword []byte
	var ok bool
	if rancherAdminPassword, ok = secret.Data["password"]; !ok {
		log.Error(fmt.Sprintf("Error getting rancher admin credentials: %v", err))
		return ""
	}

	rancherLoginURL := fmt.Sprintf("%s/%s", rancherURL, "v3-public/localProviders/local?action=login")
	payload := `{"Username": "admin", "Password": "` + string(rancherAdminPassword) + `"}`
	response, err := httpClient.Post(rancherLoginURL, "application/json", strings.NewReader(payload))
	if err != nil {
		log.Error(fmt.Sprintf("Error getting rancher admin token: %v", err))
		return ""
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		log.Errorf("Invalid response code when fetching Rancher token: %v", err)
		return ""
	}

	defer response.Body.Close()

	// extract the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read Rancher token response: %v", err)
		return ""
	}

	token, err := httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "token", "unable to find token in Rancher response")
	if err != nil {
		log.Errorf("Failed to extra token from Rancher response: %v", err)
		return ""
	}

	return token
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
		api := EventuallyGetAPIEndpoint(kubeconfigPath)
		keycloakURL := EventuallyGetURLForIngress(log, api, "keycloak", "keycloak", "https")
		rancherURL := EventuallyGetURLForIngress(log, api, "cattle-system", "rancher", "https")
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
