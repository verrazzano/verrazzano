// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

// VerifyArgoCDAccess verifies that Argocd is accessible.
func VerifyArgoCDAccess(log *zap.SugaredLogger, kubeconfigPath string) error {
	var err error

	api := EventuallyGetAPIEndpoint(kubeconfigPath)
	argocdURL := EventuallyGetURLForIngress(log, api, "argocd", "argocd-server", "https")
	httpClient := EventuallyVerrazzanoRetryableHTTPClient()
	var httpResponse *HTTPResponse

	gomega.Eventually(func() (*HTTPResponse, error) {
		httpResponse, err = GetWebPageWithClient(httpClient, argocdURL, "")
		return httpResponse, err
	}, waitTimeout, pollingInterval).Should(HasStatus(http.StatusOK))

	gomega.Expect(CheckNoServerHeader(httpResponse)).To(gomega.BeTrue(), "Found unexpected server header in response")
	return nil
}

func VerifyArgocdApplicationAccess(log *zap.SugaredLogger, kubeConfigPath string) error {
	var err error

	argocdAdminPassword, err := eventuallyGetArgocdAdminPassword(log)
	if err != nil {
		return err
	}
	httpClient, err := GetVerrazzanoHTTPClient(kubeConfigPath)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting argocd admin password: %v", err))
		return err
	}

	api := EventuallyGetAPIEndpoint(kubeConfigPath)
	argocdURL := EventuallyGetURLForIngress(log, api, "argocd", "argocd-server", "https")

	token, err := getArgoCDUserToken(log, argocdURL, "admin", string(argocdAdminPassword), httpClient)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting user token from Argocd: %v", err))
		return err
	}
	var emptyList bool
	gomega.Eventually(func() (bool, error) {
		contains, err := GetApplicationsWithClient(log, argocdURL, token)
		emptyList = contains
		return emptyList, err
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue())

	gomega.Expect(emptyList).To(gomega.BeTrue(), "Argocd UI is accessible and no applications are deployed")
	return nil
}

func eventuallyGetArgocdAdminPassword(log *zap.SugaredLogger) (string, error) {
	var err error
	var secret *corev1.Secret
	gomega.Eventually(func() error {
		secret, err = GetSecret("argocd", "argocd-initial-admin-secret")
		if err != nil {
			log.Error(fmt.Sprintf("Error getting argocd-initial-admin-secret, retrying: %v", err))
		}
		return err
	}, waitTimeout, pollingInterval).Should(gomega.BeNil())

	if secret == nil {
		return "", fmt.Errorf("Unable to get argocd admin secret")
	}

	var argocdAdminPassword []byte
	var ok bool
	if argocdAdminPassword, ok = secret.Data["password"]; !ok {
		return "", fmt.Errorf("Error getting argocd admin credentials")
	}

	return string(argocdAdminPassword), nil
}

func getArgoCDUserToken(log *zap.SugaredLogger, argoCDURL string, username string, password string, httpClient *retryablehttp.Client) (string, error) {
	argoCDLoginURL := fmt.Sprintf("%s/%s", argoCDURL, "api/v1/session")
	payload := `{"Username": "` + username + `", "Password": "` + password + `"}`
	response, err := httpClient.Post(argoCDLoginURL, "application/json", strings.NewReader(payload))
	if err != nil {
		log.Error(fmt.Sprintf("Error getting argocd admin token: %v", err))
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		log.Errorf("Invalid response code when fetching argocd token: %v", err)
		return "", err
	}

	defer response.Body.Close()

	// extract the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read argocd token response: %v", err)
		return "", err
	}

	token, err := httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "token", "unable to find token in Argocd response")
	if err != nil {
		log.Errorf("Failed to extra token from argocd response: %v", err)
		return "", err
	}

	return token, nil
}

func GetApplicationsWithClient(log *zap.SugaredLogger, argoCDURL string, token string) (bool, error) {
	argoCDLoginURL := fmt.Sprintf("%s/%s", argoCDURL, "api/v1/applications")
	client := &http.Client{}
	var bearer = "Bearer " + token

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} /* #nosec G402 */
	req, err := http.NewRequest("GET", argoCDLoginURL, nil)
	if err != nil {
		return false, err
	}

	req.Header.Add("Authorization", bearer)
	response, err := client.Do(req)
	if err != nil {
		return false, err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		log.Errorf("Invalid response code when fetching argocd token: %v", err)
		return false, err
	}

	defer response.Body.Close()

	// extract the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Failed to read argocd  response: %v", err)
		return false, err
	}

	token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "metadata", "unable to find metadata in Argocd response")
	if err != nil {
		log.Errorf("Failed to extract token from argocd response: %v", err)
		return false, err
	}

	exists := strings.Contains(token, "resourceVersion")
	return exists, nil

}
