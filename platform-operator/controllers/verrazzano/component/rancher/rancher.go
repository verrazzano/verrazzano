// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants for Kubernetes resource names
const (
	// Name is the name of the component
	Name = "rancher"
	// CattleSystem is the namespace of the component
	CattleSystem           = "cattle-system"
	IngressCAName          = "tls-rancher-ingress"
	AdminSecret            = "rancher-admin-secret"
	CACert                 = "ca.crt"
	OperatorNamespace      = "rancher-operator-system"
	defaultSecretNamespace = "cert-manager"
	namespaceLabelKey      = "verrazzano.io/namespace"
	rancherTLSSecretName   = "tls-ca"
	defaultVerrazzanoName  = "verrazzano-ca-certificate-secret"
)

// Helm Chart setter keys
const (
	ingressTLSSourceKey     = "ingress.tls.source"
	additionalTrustedCAsKey = "additionalTrustedCAs"
	privateCAKey            = "privateCA"

	// Rancher registry Keys
	useBundledSystemChartKey = "useBundledSystemChart"
	systemDefaultRegistryKey = "systemDefaultRegistry"

	// LE Keys
	letsEncryptIngressClassKey = "letsEncrypt.ingress.class"
	letsEncryptEmailKey        = "letsEncrypt.email"
	letsEncryptEnvironmentKey  = "letsEncrypt.environment"
)

const (
	letsEncryptTLSSource       = "letsEncrypt"
	caTLSSource                = "secret"
	caAdditionalPem            = "ca-additional.pem"
	privateCAValue             = "true"
	useBundledSystemChartValue = "true"
)

// Rancher HTTPS Configuration
const (
	contentTypeHeader   = "Content-Type"
	authorizationHeader = "Authorization"
	applicationJSON     = "application/json"
	// Path to get a login token
	loginActionPath = "/v3-public/localProviders/local?action=login"
	// Template body to POST for a login token
	loginActionTmpl = `
{
  "Username": "admin",
  "Password": "%s"
}
`
	// Path to get an access token
	tokPath = "/v3/token"
	// Body to POST for an access token (login token should be Bearer token)
	tokPostBody = `
{
  "type": "token",
  "description": "automation"
}`
	// Path to update server URL, as in PUT during PostInstall
	serverURLPath = "/v3/settings/server-url"
	// Template body to PUT a new server url
	serverURLTmpl = `
{
  "name": "server-url",
  "value": "https://%s"
}`
)

type (
	// restClientConfigSig is a provider for a k8s rest client implementation
	// override for unit testing
	restClientConfigSig func() (*rest.Config, rest.Interface, error)
	// httpDoSig provides a HTTP Client wrapper function for unit testing
	httpDoSig func(hc *http.Client, req *http.Request) (*http.Response, error)
	// TokenResponse is the response format Rancher uses when sending tokens in HTTP responses
	TokenResponse struct {
		Token string `json:"token"`
	}
)

// httpDo is the default HTTP Client wrapper implementation
var httpDo httpDoSig = func(hc *http.Client, req *http.Request) (*http.Response, error) {
	return hc.Do(req)
}

var restClientConfig restClientConfigSig = func() (*rest.Config, rest.Interface, error) {
	cfg, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return cfg, client.CoreV1().RESTClient(), nil
}

// For unit testing
func setRestClientConfig(f restClientConfigSig) {
	restClientConfig = f
}

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme.Environment != "production"
}

func getRancherHostname(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := nginx.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	rancherHostname := fmt.Sprintf("%s.%s.%s", Name, vz.Spec.EnvironmentName, dnsSuffix)
	return rancherHostname, nil
}
