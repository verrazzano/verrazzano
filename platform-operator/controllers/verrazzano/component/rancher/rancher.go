// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/common"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants for Kubernetes resource names
const (
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

type (
	// restClientConfigSig is a provider for a k8s rest client implementation
	// override for unit testing
	restClientConfigSig func() (*rest.Config, rest.Interface, error)
)

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
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	rancherHostname := fmt.Sprintf("%s.%s.%s", common.RancherName, vz.Spec.EnvironmentName, dnsSuffix)
	return rancherHostname, nil
}
