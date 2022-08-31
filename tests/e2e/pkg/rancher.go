// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	body, err := ioutil.ReadAll(response.Body)
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

//VerifyRancherAccess verifies that Rancher is accessible.
func VerifyRancherAccess() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
		t.Fail(err.Error())
	}

	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	rancherURL := pkg.EventuallyGetURLForIngress(t.Logs, api, "cattle-system", "rancher", "https")
	httpClient := pkg.EventuallyVerrazzanoRetryableHTTPClient()
	var httpResponse *pkg.HTTPResponse

	Eventually(func() (*pkg.HTTPResponse, error) {
		httpResponse, err = pkg.GetWebPageWithClient(httpClient, rancherURL, "")
		return httpResponse, err
	}, waitTimeout, pollingInterval).Should(pkg.HasStatus(http.StatusOK))

	Expect(pkg.CheckNoServerHeader(httpResponse)).To(BeTrue(), "Found unexpected server header in response")
}

//VerifyRancherKeycloakAuthConfig verifies that Rancher/Keycloak AuthConfig is correctly populated
func VerifyRancherKeycloakAuthConfig() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.Logs.Error(fmt.Sprintf("Error getting kubeconfig: %v", err))
		t.Fail(err.Error())
	}

	start := time.Now()
	t.Logs.Info("Verify Keycloak AuthConfig")
	api := pkg.EventuallyGetAPIEndpoint(kubeconfigPath)
	keycloakURL := pkg.EventuallyGetURLForIngress(t.Logs, api, "keycloak", "keycloak", "https")
	rancherURL := pkg.EventuallyGetURLForIngress(t.Logs, api, "cattle-system", "rancher", "https")
	k8sClient, err := pkg.GetDynamicClientInCluster(kubeconfigPath)
	if err != nil {
		t.Logs.Error(fmt.Sprintf("Error getting K8S client: %v", err))
		t.Fail(err.Error())
	}

	Eventually(func() (bool, error) {
		authConfigData, err := k8sClient.Resource(gvkToGvr(common.GVKAuthConfig)).Get(context.Background(), common.AuthConfigKeycloak, v1.GetOptions{})
		if err != nil {
			t.Logs.Error(fmt.Sprintf("error getting keycloak oidc authConfig: %v", err))
			return false, err
		}

		authConfigAttributes := authConfigData.UnstructuredContent()
		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeAccessMode, authConfigAttributes[rancher.AuthConfigKeycloakAttributeAccessMode].(string), rancher.AuthConfigKeycloakAccessMode); err != nil {
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeClientID, authConfigAttributes[rancher.AuthConfigKeycloakAttributeClientID].(string), rancher.AuthConfigKeycloakClientIDRancher); err != nil {
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeGroupSearchEnabled, authConfigAttributes[rancher.AuthConfigKeycloakAttributeGroupSearchEnabled].(bool), true); err != nil {
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeAuthEndpoint, authConfigAttributes[rancher.AuthConfigKeycloakAttributeAuthEndpoint].(string), keycloakURL+rancher.AuthConfigKeycloakURLPathAuthEndPoint); err != nil {
			return false, err
		}

		if err = verifyAuthConfigAttribute(rancher.AuthConfigKeycloakAttributeRancherURL, authConfigAttributes[rancher.AuthConfigKeycloakAttributeRancherURL].(string), rancherURL+rancher.AuthConfigKeycloakURLPathVerifyAuth); err != nil {
			return false, err
		}

		authConfigClientSecret := authConfigAttributes[common.AuthConfigKeycloakAttributeClientSecret].(string)
		if authConfigClientSecret == "" {
			err = fmt.Errorf("keycloak auth config attribute %s not correctly configured, value is empty", common.AuthConfigKeycloakAttributeClientSecret)
			t.Logs.Error(err.Error())
			return false, err
		}

		return true, nil
	}, waitTimeout, pollingInterval).Should(Equal(true), "keycloak oidc authconfig not configured correctly")
	metrics.Emit(t.Metrics.With("get_kc_authconfig_state_elapsed_time", time.Since(start).Milliseconds()))
}
