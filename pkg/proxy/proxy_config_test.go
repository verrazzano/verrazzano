// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yaml "sigs.k8s.io/yaml"
)

// const dnsSuffix = "default.example.com"
const dnsSuffix = "default.my.domain.nip.io"

func getProxyBaseConfig() OidcProxyConfig {
	proxyConfig := OidcProxyConfig{}

	proxyConfig.OidcRealm = OidcRealmName
	proxyConfig.PKCEClientID = OidcPkceClientID
	proxyConfig.PGClientID = OidcPgClientID
	proxyConfig.RequiredRealmRole = OidcRequiredRealmRole

	proxyConfig.OidcProviderHost = "keycloak." + dnsSuffix
	proxyConfig.OidcProviderHostInCluster = "keycloak-http.keycloak.svc.cluster.local"

	return proxyConfig
}

// getProxyConfigWithParams returns an OidcProxyConfig struct
func getProxyConfigWithParams(keycloakURL string) OidcProxyConfig {
	proxyConfig := getProxyBaseConfig()

	proxyConfig.OidcCallbackPath = OidcCallbackPath
	proxyConfig.OidcLogoutCallbackPath = OidcLogoutCallbackPath
	proxyConfig.AuthnStateTTL = OidcAuthnStateTTL

	// if keycloakURL is present, meanning it is a managed cluster, keycloakURL is the admin cluster's keycloak url
	if len(keycloakURL) > 0 {
		u, err := url.Parse(keycloakURL)
		if err == nil {
			proxyConfig.OidcProviderHost = u.Host
			proxyConfig.OidcProviderHostInCluster = ""
		}
	}

	return proxyConfig
}

func getProxyConfig() OidcProxyConfig {
	return getProxyConfigWithParams("")
}

func generateConfigmapYaml(name, namespace string, labels, annotations, configs map[string]string) string {
	yaml := "---"
	yaml = fmt.Sprintf("%s\napiVersion: v1", yaml)
	yaml = fmt.Sprintf("%s\nkind: ConfigMap", yaml)
	yaml = fmt.Sprintf("%s\nmetadata:", yaml)
	yaml = fmt.Sprintf("%s\n  name: %s", yaml, name)
	yaml = fmt.Sprintf("%s\n  namespace: %s", yaml, namespace)
	yaml = fmt.Sprintf("%s\n  labels:", yaml)
	for key, value := range labels {
		yaml = fmt.Sprintf("%s\n    %s: %s", yaml, key, value)
	}
	yaml = fmt.Sprintf("%s\n  annotations:", yaml)
	for key, value := range annotations {
		yaml = fmt.Sprintf("%s\n    %s: %s", yaml, key, value)
	}
	yaml = fmt.Sprintf("%s\ndata:\n", yaml)
	for key, value := range configs {
		yaml = fmt.Sprintf("%s  %s: |\n    %s\n", yaml, key, value)
	}
	return yaml
}

func generateConfigmap(name, namespace string, labels, configs map[string]string) (corev1.ConfigMap, error) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: configs,
	}
	return configMap, nil
}

func checkConfigMapData(data map[string]string, expectedCount int, config OidcProxyConfig) error {
	if len(data) != expectedCount {
		return fmt.Errorf("Wrong number of keys in the map (expected %v, got %v", expectedCount, len(data))
	}
	for file, content := range data {
		if file == "" {
			return fmt.Errorf("Nil or empty filename")
		}
		if content == "" {
			return fmt.Errorf("Nil or empty file content")
		}
		if file == "auth.lua" {
			expected := fmt.Sprintf(`local oidcRealm = "%s"`, config.OidcRealm)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local oidcClient = "%s"`, config.PKCEClientID)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local oidcDirectAccessClient = "%s"`, config.PGClientID)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local requiredRole = "%s"`, config.RequiredRealmRole)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local authStateTtlInSec = tonumber("%d")`, config.AuthnStateTTL)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local oidcProviderHost = "%s"`, config.OidcProviderHost)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local oidcProviderHostInCluster = "%s"`, config.OidcProviderHostInCluster)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
		} else if file == "conf.lua" {
			expected := fmt.Sprintf(`local callbackPath = "%s"`, config.OidcCallbackPath)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local logoutCallbackPath = "%s"`, config.OidcLogoutCallbackPath)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
		}
	}
	return nil
}

func checkForValidYaml(configMapYaml string) error {
	configMap := corev1.ConfigMap{}
	err := yaml.Unmarshal([]byte(configMapYaml), &configMap)
	if err != nil {
		return err
	}
	if strings.Contains(configMapYaml, "{{") {
		return fmt.Errorf("Found unevaluated template actions in yaml")
	}
	return nil
}

func TestGetConfigmapData(t *testing.T) {
	config := getProxyConfig()
	data, err := GetProxyConfigmapData(config)
	if err != nil {
		t.Fatalf("Error getting proxy config map data: %v", err)
	}
	err = checkConfigMapData(data, 5, config)
	if err != nil {
		t.Fatalf("Invalid map data: %v", err)
	}
	t.Log("Got expected config map data")
}

func TestCreateProxyConfigmapYaml(t *testing.T) {
	data, err := GetProxyConfigmapData(getProxyConfig())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	labels := map[string]string{
		"app":                          "verrazzano-api",
		"app.kubernetes.io/managed-by": "Helm",
	}
	annotations := map[string]string{
		"meta.helm.sh/release-name":      "verrazzano",
		"meta.helm.sh/release-namespace": "verrazzano-system",
	}
	config := generateConfigmapYaml("api-nginx-conf", "verrazzano-system", labels, annotations, data)
	err = checkForValidYaml(config)
	if err != nil {
		t.Fatalf("Invalid yaml for configmap: %v", err)
	}
	t.Logf("Got expected config map yaml: %p", &config)
	//t.Logf("\n%s", config)
}

func TestCreateProxyConfigmap(t *testing.T) {
	data, err := GetProxyConfigmapData(getProxyConfig())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	configMap, err := generateConfigmap("testConfigMap", "testConfigMapNamespace", nil, data)
	if err != nil {
		t.Fatalf("Invalid configmap for proxy: %v", err)
	}
	t.Logf("Got expected configmap: %p", &configMap)
	//t.Logf("\n%v", configMap)
}
