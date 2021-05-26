package proxy

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getGenericProxyConfig() OidcProxyConfig {
	proxyConfig := OidcProxyConfig{}

	proxyConfig.StartupDir = "`dirname $0`"

	proxyConfig.SSLVerifyOptions = OidcSSLVerifyOptions
	proxyConfig.SSLTrustedCAOptions = OidcSSLTrustedOptions
	proxyConfig.Host = "localhost"
	proxyConfig.Port = 8443

	proxyConfig.Ingress = "foo.bar.com"
	proxyConfig.OidcProviderHost = "keycloak.foo.bar.com"
	proxyConfig.OidcProviderHostInCluster = "keycloak-http.keycloak.svc.cluster.local"

	return proxyConfig
}

func TestGetConfigMapData(t *testing.T) {
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

func TestCreateConfigmap(t *testing.T) {
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
