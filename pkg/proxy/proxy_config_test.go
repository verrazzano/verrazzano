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

func getProxyConfigAPIProxyWithParams(providerHost string) OidcProxyConfig {
	proxyConfig := OidcProxyConfig{}

	proxyConfig.Mode = ProxyModeAPI
	proxyConfig.OidcRealm = OidcRealmName
	proxyConfig.PKCEClientID = OidcPkceClientID
	proxyConfig.PGClientID = OidcPgClientID
	proxyConfig.RequiredRealmRole = OidcRequiredRealmRole

	proxyConfig.OidcProviderHost = providerHost
	proxyConfig.OidcProviderHostInCluster = "keycloak-http.keycloak.svc.cluster.local"

	return proxyConfig
}

func getProxyConfigAPIProxy() OidcProxyConfig {
	return getProxyConfigAPIProxyWithParams("keycloak.default.example.com")
}

// getProxyConfigOidcProxy returns an OidcProxyConfig struct
func getProxyConfigOidcProxyWithParams(ingressHost, verrazzanoURI, keycloakURL string, backendPort int, SSLEnabled bool) OidcProxyConfig {
	proxyConfig := OidcProxyConfig{}

	proxyConfig.Mode = ProxyModeOauth
	proxyConfig.OidcRealm = OidcRealmName
	proxyConfig.PKCEClientID = OidcPkceClientID
	proxyConfig.PGClientID = OidcPgClientID
	proxyConfig.RequiredRealmRole = OidcRequiredRealmRole

	proxyConfig.OidcCallbackPath = OidcCallbackPath
	proxyConfig.OidcLogoutCallbackPath = OidcLogoutCallbackPath
	proxyConfig.AuthnStateTTL = OidcAuthnStateTTL

	proxyConfig.Host = "localhost"
	proxyConfig.Port = backendPort
	proxyConfig.SSLEnabled = SSLEnabled

	proxyConfig.Ingress = ingressHost

	proxyConfig.OidcProviderHost = fmt.Sprintf("%s.%s", "keycloak", verrazzanoURI)
	proxyConfig.OidcProviderHostInCluster = "keycloak-http.keycloak.svc.cluster.local"

	// if keycloakURL is present, meanning it is a managed cluster, keycloakURL is the admin keycloak url
	if len(keycloakURL) > 0 {
		u, err := url.Parse(keycloakURL)
		if err == nil {
			proxyConfig.OidcProviderHost = u.Host
			proxyConfig.OidcProviderHostInCluster = ""
		}
	}

	return proxyConfig
}

func getProxyConfigOidcProxy() OidcProxyConfig {
	return getProxyConfigOidcProxyWithParams(
		"grafana.vmi.system.default.111.222.333.444.nip.io",
		"default.111.222.333.444.nip.io",
		"",
		9000,
		false,
	)
}

func generateConfigmapYaml(name, namespace string, labels, configs map[string]string) string {
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
	yaml = fmt.Sprintf("%s\ndata:\n", yaml)
	for key, value := range configs {
		yaml = fmt.Sprintf("%s  %s: |\n%s\n", yaml, key, value)
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
			expected := fmt.Sprintf(`local ingressUri = 'https://'..'%s'`, config.Ingress)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local callbackPath = "%s"`, config.OidcCallbackPath)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
			expected = fmt.Sprintf(`local logoutCallbackPath = "%s"`, config.OidcLogoutCallbackPath)
			if !strings.Contains(content, expected) {
				return fmt.Errorf("File %s, did not find expected substitution '%s'", file, expected)
			}
		} else if file == "nginx.conf" && config.Mode == ProxyModeOauth {
			expected := fmt.Sprintf(`server %s:%v fail_timeout=30s max_fails=10;`, config.Host, config.Port)
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

func TestGetConfigMapDataAPIProxy(t *testing.T) {
	config := getProxyConfigAPIProxy()
	data, err := GetOidcProxyConfigMapData(config)
	if err != nil {
		t.Fatalf("Error getting proxy config map data: %v", err)
	}
	err = checkConfigMapData(data, 5, config)
	if err != nil {
		t.Fatalf("Invalid map data: %v", err)
	}
	t.Log("Got expected API config map data")
}

func TestGetConfigMapDataOidcProxy(t *testing.T) {
	config := getProxyConfigOidcProxy()
	data, err := GetOidcProxyConfigMapData(config)
	if err != nil {
		t.Fatalf("Error getting proxy config map data: %v", err)
	}
	err = checkConfigMapData(data, 4, config)
	if err != nil {
		t.Fatalf("Invalid map data: %v", err)
	}
	t.Log("Got expected Oidc config map data")
}

func TestCreateAPIProxyConfigYaml(t *testing.T) {
	data, err := GetOidcProxyConfigMapData(getProxyConfigAPIProxy())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	labels := map[string]string{
		"app": "verrazzano-api",
	}
	config := generateConfigmapYaml("api-nginx-conf", "verrazzano-system", labels, data)
	err = checkForValidYaml(config)
	if err != nil {
		t.Fatalf("Invalid yaml for API configmap: %v", err)
	}
	t.Logf("Got expected API config map yaml: %p", &config)
	//t.Logf("\n%s", config)
}

func TestCreateOidcProxyConfigYaml(t *testing.T) {
	data, err := GetOidcProxyConfigMapData(getProxyConfigOidcProxy())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	labels := map[string]string{
		"k8s-app":              "verrazzano.io",
		"vmo.v1.verrazzano.io": "system",
	}
	config := generateConfigmapYaml("vmi-system-grafana-oidc-config", "verrazzano-system", labels, data)
	err = checkForValidYaml(config)
	if err != nil {
		t.Fatalf("Invalid yaml for Oidc configmap: %v", err)
	}
	t.Logf("Got expected Oidc config map yaml: %p", &config)
	//t.Logf("\n%s", config)
}

func TestCreateAPIProxyConfigmap(t *testing.T) {
	data, err := GetOidcProxyConfigMapData(getProxyConfigAPIProxy())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	configMap, err := generateConfigmap("testConfigMap", "testConfigMapNamespace", nil, data)
	if err != nil {
		t.Fatalf("Invalid configmap for API proxy: %v", err)
	}
	t.Logf("Got expected Oidc configmap: %p", &configMap)
	//t.Logf("\n%v", configMap)
}

func TestCreateOidcProxyConfigmap(t *testing.T) {
	data, err := GetOidcProxyConfigMapData(getProxyConfigOidcProxy())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	configMap, err := generateConfigmap("testConfigMap", "testConfigMapNamespace", nil, data)
	if err != nil {
		t.Fatalf("Invalid configmap for Oidc proxy: %v", err)
	}
	t.Logf("Got expected Oidc configmap: %p", &configMap)
	//t.Logf("\n%v", configMap)
}
