// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
)

// Constants for Kubernetes resource names
const (
	// ComponentName is the name of the component
	ComponentName = "rancher"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace     = "cattle-system"
	defaultSecretNamespace = "cert-manager"
	namespaceLabelKey      = "verrazzano.io/namespace"
	adminSecretName        = "rancher-admin-secret"
	rancherTLSSecretName   = "tls-ca"
	defaultVerrazzanoName  = "verrazzano-ca-certificate-secret"
	clusterAgentDeployName = "cattle-cluster-agent"
	nodeAgentDaemonsetName = "cattle-node-agent"
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
	letsEncryptTLSSource = "letsEncrypt"
	caTLSSource          = "secret"
)

type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme.Environment != "production"
}
