// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/gomega"
	"go.uber.org/zap"

	"gopkg.in/yaml.v2"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func VerifySystemVMIComponent(log *zap.SugaredLogger, api *APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *UsernamePassword, ingressName, expectedURLPrefix string) bool {
	ingress, err := getIngress(ingressName)
	if err != nil {
		log.Errorf("Error getting ingress: %v", err)
		return false
	}
	vmiComponentURL := fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
	if !strings.HasPrefix(vmiComponentURL, expectedURLPrefix) {
		log.Errorf("URL '%s' does not have expected prefix: %s", vmiComponentURL, expectedURLPrefix)
		return false
	}
	return AssertURLAccessibleAndAuthorized(sysVmiHTTPClient, vmiComponentURL, vmiCredentials)
}

func getIngress(ingressName string) (*netv1.Ingress, error) {
	ingressList, err := GetIngressList("verrazzano-system")
	if err != nil {
		return nil, err
	}
	for _, ingress := range ingressList.Items {
		if ingress.Name == ingressName {
			return &ingress, nil
		}
	}

	return nil, errors.NewNotFound(schema.GroupResource{}, ingressName)
}

func VerifyOpenSearchComponent(log *zap.SugaredLogger, api *APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *UsernamePassword) bool {
	return VerifySystemVMIComponent(log, api, sysVmiHTTPClient, vmiCredentials, "vmi-system-es-ingest", "https://elasticsearch.vmi.system")
}

func VerifyOpenSearchDashboardsComponent(log *zap.SugaredLogger, api *APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *UsernamePassword) bool {
	return VerifySystemVMIComponent(log, api, sysVmiHTTPClient, vmiCredentials, "vmi-system-kibana", "https://kibana.vmi.system")
}

func VerifyPrometheusComponent(log *zap.SugaredLogger, api *APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *UsernamePassword) bool {
	return VerifySystemVMIComponent(log, api, sysVmiHTTPClient, vmiCredentials, "vmi-system-prometheus", "https://prometheus.vmi.system")
}

func VerifyGrafanaComponent(log *zap.SugaredLogger, api *APIEndpoint, sysVmiHTTPClient *retryablehttp.Client, vmiCredentials *UsernamePassword) bool {
	return VerifySystemVMIComponent(log, api, sysVmiHTTPClient, vmiCredentials, "vmi-system-grafana", "https://grafana.vmi.system")
}

func EventuallyGetSystemVMICredentials() *UsernamePassword {
	var vmiCredentials *UsernamePassword
	gomega.Eventually(func() (*UsernamePassword, error) {
		var err error
		vmiCredentials, err = GetSystemVMICredentials()
		return vmiCredentials, err
	}, waitTimeout, pollingInterval).ShouldNot(gomega.BeNil())
	return vmiCredentials
}

// GetSystemVMICredentials - Obtain VMI system credentials
func GetSystemVMICredentials() (*UsernamePassword, error) {
	secret, err := GetSecret("verrazzano-system", "verrazzano")
	if err != nil {
		return nil, err
	}

	username := secret.Data["username"]
	password := secret.Data["password"]
	if username == nil || password == nil {
		return nil, fmt.Errorf("username and password fields required in secret %v", secret)
	}

	return &UsernamePassword{
		Username: string(username),
		Password: string(password),
	}, nil
}

// GetPrometheusConfig - Returns the Prometehus Configmap, Marshalled prometehus.yml and the scrape config list
func GetPrometheusConfig() (*v1.ConfigMap, []interface{}, map[interface{}]interface{}, error) {
	configMap, err := GetConfigMap(vzconst.VmiPromConfigName, vzconst.VerrazzanoSystemNamespace)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed getting configmap: %v", err))
		return nil, nil, nil, err
	}

	prometheusConfig := configMap.Data["prometheus.yml"]
	var configYaml map[interface{}]interface{}
	err = yaml.Unmarshal([]byte(prometheusConfig), &configYaml)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed getting configmap yaml: %v", err))
		return nil, nil, nil, err
	}

	scrapeConfigsData := configYaml["scrape_configs"]
	scrapeConfigs := scrapeConfigsData.([]interface{})
	return configMap, scrapeConfigs, configYaml, nil
}
