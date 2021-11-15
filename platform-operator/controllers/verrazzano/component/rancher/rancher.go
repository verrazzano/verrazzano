// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
	"os/exec"
	"strings"
)

// ComponentName is the name of the component
const (
	ComponentName      = "rancher"
	ComponentNamespace = "cattle-system"
	NamespaceLabelKey  = "verrazzano.io/namespace"
	AdminSecret        = "rancher-admin-secret"
	RancherTlsSecret   = "tls-ca"
	clusterAgentDeploy = "cattle-cluster-agent"
	nodeAgentDaemonset = "cattle-node-agent"
	// Extra Helm chart arguments for LetsEncrypt

	extraRancherArgs = "--set letsEncrypt.ingress.class=rancher --set letsEncrypt.email=%s --set letsEncrypt.environment=%s --set additionalTrustedCAs=%s"
	// Patch data for LetsEncrypt
	rancherPatchData = "{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"%v auth\",\"external-dns.alpha.kubernetes.io/target\":\"%v\",\"cert-manager.io/issuer\":null,\"cert-manager.io/issuer-kind\":null,\"external-dns.alpha.kubernetes.io/ttl\":\"60\"}}}"

	// CA
	//RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${NAME}.${DNS_SUFFIX} auth\",\"cert-manager.io/cluster-issuer\":\"verrazzano-cluster-issuer\"}}}"
	// LE
	//RANCHER_PATCH_DATA="{\"metadata\":{\"annotations\":{\"kubernetes.io/tls-acme\":\"true\",\"nginx.ingress.kubernetes.io/auth-realm\":\"${DNS_SUFFIX} auth\",\"external-dns.alpha.kubernetes.io/target\":\"verrazzano-ingress.${NAME}.${DNS_SUFFIX}\",\"cert-manager.io/issuer\":null,\"cert-manager.io/issuer-kind\":null,\"external-dns.alpha.kubernetes.io/ttl\":\"60\"}}}"
)

// Helm Chart keys
const (
	ingressTLSSourceKey         = "ingress.tls.source"
	hostnameKey                 = "rancher"
	additionalTrustedCAsKey     = "additionalTrustedCAs"
	defaultVerrazzanoSecretName = "verrazzano-ca-certificate-secret"
	defaultSecretNamespace      = "cert-manager"
	privateCAKey                = "privateCA"

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

var (
	runner = vzos.DefaultRunner{}
)

func run(log *zap.SugaredLogger, command string, args string) (stdout, stderr string, err error) {
	cmd := exec.Command(command, strings.Split(args, " ")...)
	log.Infof("Running command: %s", cmd.String())
	stdoutBytes, stderrBytes, err := runner.Run(cmd)
	return string(stdoutBytes), string(stderrBytes), err
}

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme.Environment != "production"
}
