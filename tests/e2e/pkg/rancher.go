// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"go.uber.org/zap"
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
