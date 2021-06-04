// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package proxy

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getGenericProxyConfig() OidcProxyConfig {
	proxyConfig := OidcProxyConfig{}

	proxyConfig.Host = "localhost"
	proxyConfig.Port = 8443

	proxyConfig.Ingress = "foo.bar.com"
	proxyConfig.OidcProviderHost = "keycloak.foo.bar.com"
	proxyConfig.OidcProviderHostInCluster = "keycloak-http.keycloak.svc.cluster.local"

	return proxyConfig
}

func getAPIProxyConfig() OidcProxyConfig {
	proxyConfig := OidcProxyConfig{}

	proxyConfig.Mode = "api-proxy"
	proxyConfig.OidcRealm = "verrazzano-system"
	proxyConfig.PKCEClientID = "verrazzano-pkce"
	proxyConfig.PGClientID = "verrazzano-pg"
	proxyConfig.RequiredRealmRole = "vz_api_access"

	proxyConfig.OidcProviderHost = "keycloak.will.129.159.243.176.nip.io"
	proxyConfig.OidcProviderHostInCluster = "keycloak-http.keycloak.svc.cluster.local"

	return proxyConfig
}

func NotesGetConfigMapData(t *testing.T) {
	//var result map[string]string
	result, err := GetOidcProxyConfigMapData(getGenericProxyConfig())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	if len(result) != 4 {
		t.Fatalf("Wrong number of keys in the map: %v", len(result))
	}
	for file, content := range result {
		if file == "" || content == "" {
			t.Fatalf("empty key or content in map")
		}
		t.Logf("\n%s: %s\n", file, content)
	}
}

func NotesCreateConfigmap(t *testing.T) {
	data, err := GetOidcProxyConfigMapData(getGenericProxyConfig())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testConfigMap",
			Namespace: "testConfigMapNamespace",
		},
		Data: data,
	}
	t.Logf("\n%s", configMap)
}

func TestCreateAPIProxyConfigMap(t *testing.T) {
	data, err := GetOidcProxyConfigMapData(getAPIProxyConfig())
	if err != nil {
		t.Fatalf("Error getting config map data: %v", err)
	}

	config := "---"
	config = fmt.Sprintf("%s\napiVersion: v1", config)
	config = fmt.Sprintf("%s\nkind: ConfigMap", config)
	config = fmt.Sprintf("%s\nmetadata:", config)
	config = fmt.Sprintf("%s\n  name: api-nginx-conf", config)
	config = fmt.Sprintf("%s\n  namespace: verrazzano-system", config)
	config = fmt.Sprintf("%s\n  labels:", config)
	config = fmt.Sprintf("%s\n    app: verrazzano-api", config)
	config = fmt.Sprintf("%s\ndata:\n", config)
	for key, value := range data {
		config = fmt.Sprintf("%s  %s: %s\n", config, key, value)
	}
	t.Logf(config)
}
