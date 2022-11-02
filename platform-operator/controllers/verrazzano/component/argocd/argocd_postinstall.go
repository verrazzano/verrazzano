// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patchArgoCDSecret
func patchArgoCDSecret(ctx spi.ComponentContext) error {
	clientSecret, err := keycloak.GetArgoCDClientSecretFromKeycloak(ctx)
	if err != nil {
		return ctx.Log().ErrorfNewErr("failed configuring keycloak as OIDC provider for argocd, unable to fetch argocd client secret: %s", err)
	}

	// update the secret with the updated client secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-secret",
			Namespace: constants.ArgoCDNamespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		secret.Data["oidc.keycloak.clientSecret"] = []byte(clientSecret)
		return nil
	}); err != nil {
		return err
	}

	ctx.Log().Debugf("patchArgoCDSecret: ArgoCD secret operation result: %v", err)
	return err
}

// patchArgoCDConfigMap
func patchArgoCDConfigMap(ctx spi.ComponentContext) error {
	argocdHostName, err := getArgoCDHostname(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting ArgoCD host name: %v", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}

	var oidcString = fmt.Sprintf(`|
        name: Keycloak
        issuer: "https://%s/%s"
        clientID: argocd
        clientSecret: $oidc.keycloak.clientSecret
        requestedScopes: ["openid", "profile", "email", "groups"]`, argocdHostName, "auth/realms/verrazzano-system")

	// Add the oidc configuration to enable our keycloak authentication.
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["url"] = fmt.Sprintf("https://%s", argocdHostName)
		cm.Data["oidc.config"] = oidcString
		return nil
	}); err != nil {
		return err
	}

	ctx.Log().Debugf("patchArgoCDConfigMap: ArgoCD cm operation result: %v", err)
	return err
}

// patchArgoCDConfigMap
func patchArgoCDRbacConfigMap(ctx spi.ComponentContext) error {
	rbaccm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-rbac-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}

	var policyString = `|
        g, verrazzano-admins, role:admin`
	var err error

	// Disable the built-in admin user. Grant admin (role:admin) to verrazzano-admins group
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), rbaccm, func() error {
		if rbaccm.Data == nil {
			rbaccm.Data = make(map[string]string)
		}
		rbaccm.Data["policy.csv"] = policyString
		return nil
	}); err != nil {
		return err
	}

	ctx.Log().Debugf("patchArgoCDRbacConfigMap: ArgoCD rbac cm operation result: %v", err)
	return err
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
	argocdHostname := fmt.Sprintf("%s.%s.%s", common.ArgoCDName, env, dnsSuffix)
	return argocdHostname, nil
}
