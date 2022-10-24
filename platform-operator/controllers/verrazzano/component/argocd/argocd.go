// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Helm Chart setter keys
const (
	ingressTLSSourceKey     = "ingress.tls.source"
	additionalTrustedCAsKey = "additionalTrustedCAs"
	privateCAKey            = "privateCA"

	// LE Keys
	letsEncryptIngressClassKey = "letsEncrypt.ingress.class"
	letsEncryptEmailKey        = "letsEncrypt.email"
	letsEncryptEnvironmentKey  = "letsEncrypt.environment"
)

const (
	letsEncryptTLSSource = "letsEncrypt"
	caTLSSource          = "secret"
	caCertsPem           = "cacerts.pem"
	caCert               = "ca.crt"
	privateCAValue       = "true"
	SettingServerURL     = "server-url"
)

// Constants for Kubernetes resource names
const (
	defaultSecretNamespace = "cert-manager"
	namespaceLabelKey      = "verrazzano.io/namespace"
	argoCDTLSSecretName    = "tls-ca"
	defaultVerrazzanoName  = "verrazzano-ca-certificate-secret"
)

//Constants f

// GetOverrides returns the install overrides from v1beta1.Verrazzano CR
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.ArgoCD != nil {
			return effectiveCR.Spec.Components.ArgoCD.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ArgoCD != nil {
			return effectiveCR.Spec.Components.ArgoCD.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func useAdditionalCAs(acme vzapi.Acme) bool {
	return acme.Environment != "production"
}

func getArgoCDHostname(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	env := vz.Spec.EnvironmentName
	if len(env) == 0 {
		env = constants.DefaultEnvironmentName
	}
	argoCDHostname := fmt.Sprintf("%s.%s.%s", common.ArgoCDName, env, dnsSuffix)
	return argoCDHostname, nil
}

// isArgoCDReady checks the state of the expected argocd deployments and returns true if they are in a ready state
func isArgoCDReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      common.ArgoCDApplicationController,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDDexServer,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDNotificationController,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDRedis,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDRepoServer,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDServer,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}
