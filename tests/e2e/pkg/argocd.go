// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

const (
	argoCdHelidonApplicationFile = "tests/e2e/config/scripts/hello-helidon-argocd-application.yaml"
)

// VerifyArgoCDAccess verifies that Argocd is accessible.
func VerifyArgoCDAccess(log *zap.SugaredLogger) error {
	var err error

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	api := EventuallyGetAPIEndpoint(kubeconfigPath)
	argocdURL := EventuallyGetURLForIngress(log, api, constants.ArgoCDNamespace, "argocd-server", "https")
	httpClient := EventuallyVerrazzanoRetryableHTTPClient()
	var httpResponse *HTTPResponse

	gomega.Eventually(func() (*HTTPResponse, error) {
		httpResponse, err = GetWebPageWithClient(httpClient, argocdURL, "")
		return httpResponse, err
	}, waitTimeout, pollingInterval).Should(HasStatus(http.StatusOK))

	gomega.Expect(CheckNoServerHeader(httpResponse)).To(gomega.BeTrue(), "Found unexpected server header in response")
	return nil
}

func VerifyArgoCDApplicationAccess(log *zap.SugaredLogger) error {
	var err error

	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
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
	argocdURL := EventuallyGetURLForIngress(log, api, constants.ArgoCDNamespace, "argocd-server", "https")

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
		secret, err = GetSecret(constants.ArgoCDNamespace, "argocd-initial-admin-secret")
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

// GetApplicationsWithClient returns true if the user is able to access the applications page post Argo CD install
func GetApplicationsWithClient(log *zap.SugaredLogger, argoCDURL string, token string) (bool, error) {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return false, err
	}

	httpClient, err := GetVerrazzanoHTTPClient(kubeConfigPath)
	if err != nil {
		log.Error(fmt.Sprintf("Error getting argocd admin password: %v", err))
		return false, err
	}

	argoCDLoginURL := fmt.Sprintf("%s/%s", argoCDURL, "api/v1/applications")
	req, err := retryablehttp.NewRequest("GET", argoCDLoginURL, nil)
	if err != nil {
		log.Error("Unexpected error while creating new request=%v", err)
		return false, err
	}
	var bearer = "Bearer " + token

	req.Header.Add("Authorization", bearer)
	response, err := httpClient.Do(req)
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

// CreateArgoCDGitApplication creates an application in Argo CD by connecting to the Git repo
// Applies the Argo CD Application to the kubernetes cluster
func CreateArgoCDGitApplication() error {
	Log(Info, "Create Argo CD Application Project")
	gomega.Eventually(func() error {
		file, err := FindTestDataFile(argoCdHelidonApplicationFile)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, "argocd")
	}, helidonWaitTimeout, helidonPollingInterval).ShouldNot(gomega.HaveOccurred(), "Failed to create Argo CD Application Project file")

	return nil
}

// This function retrieves the ArgoCD password to log into rancher, based on the provided name and namespace of a secret that holds this information
func RetrieveArgoCDPassword(namespace, name string) (string, error) {
	s, err := GetSecret(namespace, name)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return "", err
	}
	argoCDPasswordForSecret, ok := s.Data["password"]
	if !ok {
		Log(Error, fmt.Sprintf("Failed to find password value in ArgoCD secret %s in namespace %s", name, namespace))
		return "", fmt.Errorf("Failed to find password value in ArgoCD secret %s in namespace %s", name, namespace)
	}
	return string(argoCDPasswordForSecret), nil
}
