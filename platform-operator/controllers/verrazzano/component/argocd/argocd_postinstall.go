// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"encoding/base64"
	"fmt"
	"gopkg.in/yaml.v2"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
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
		secret.Data["oidc.keycloak.clientSecret"] = []byte(base64.StdEncoding.EncodeToString([]byte(clientSecret)))
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

	argocm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}

	var oidcString = fmt.Sprintf(`
    data:
      url: "https://%s"
      oidc.config: |
        name: Keycloak
        issuer: "https://%s/%s"
        clientID: argocd
        clientSecret: $oidc.keycloak.clientSecret
        requestedScopes: ["openid", "profile", "email", "groups"]`, argocdHostName, argocdHostName, "auth/realms/verrazzano-system")

	type data struct {
		Data map[string]string `yaml:"data"`
	}
	t := data{}

	// Add the oidc configuration to enable our keycloak authentication.
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), argocm, func() error {
		err := yaml.Unmarshal([]byte(oidcString), &t)
		if err != nil {
			ctx.Log().ErrorfNewErr("error: %v", err)
		}
		argocm.Data = t.Data
		return nil
	}); err != nil {
		return err
	}

	return err
}

// patchArgoCDConfigMap
func patchArgoCDRbacConfigMap(ctx spi.ComponentContext) error {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-rbac-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}

	var policyString = `
    data:
      policy.csv: |
        g, verrazzano-admins, role:admin`

	type data struct {
		Data map[string]string `yaml:"data"`
	}
	t := data{}

	// Disable the built-in admin user. Grant admin (role:admin) to verrazzano-admins group
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), configMap, func() error {
		err := yaml.Unmarshal([]byte(policyString), &t)
		if err != nil {
			ctx.Log().ErrorfNewErr("error: %v", err)
		}
        configMap.Data = t.Data
		return nil
	})
	return err
}