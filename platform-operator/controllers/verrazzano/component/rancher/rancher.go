// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants for Kubernetes resource names
const (
	// Name is the name of the component
	Name = "rancher"
	// CattleSystem is the namespace of the component
	CattleSystem           = "cattle-system"
	IngressCASecret        = "tls-rancher-ingress"
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
	privateCAValue             = "true"
	useBundledSystemChartValue = "true"
)

// Rancher HTTPS Configuration
const (
	contentTypeHeader   = "Content-Type"
	authorizationHeader = "Authorization"
	applicationJSON     = "application/json"
	loginActionPath     = "/v3-public/localProviders/local?action=login"
	loginActionTmpl     = `
{
  "Username": "admin",
  "Password": "%s"
}
`
	tokenPath = "/v3/token"
	tokenBody = `
{
  "type": "token",
  "description": "automation"
}`
	serverUrlPath = "/v3/settings/server-url"
	serverUrlTmpl = `
{
  "name": "server-url",
  "value": "https://%s"
}`
)

type (
	clientConfigSig func() (*rest.Config, rest.Interface, error)
	bashFuncSig     func(inArgs ...string) (string, string, error)

	TokenResponse struct {
		Token string `json:"token"`
	}
)

var bashFunc bashFuncSig = vzos.RunBash

var restClientConfig clientConfigSig = func() (*rest.Config, rest.Interface, error) {
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

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

// For unit testing
func setRestClientConfig(f clientConfigSig) {
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
