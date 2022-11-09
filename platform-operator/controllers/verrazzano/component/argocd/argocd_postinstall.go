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
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

//const oidcConfigTemplate = `
//name: Keycloak
//issuer: {{.KeycloakUrl}}
//clientID: argocd
//clientSecret: $oidc.keycloak.clientSecret
//requestedScopes: ["openid", "profile", "email", "groups"]
//rootCA: |
//  {{.CaCert}}
//`

type OIDCConfig struct {
	Name            string   `json:"keycloak"`
	Issuer          string   `json:"issuer"`
	ClientID        string   `json:"clientID"`
	ClientSecret    string   `json:"clientSecret"`
	RequestedScopes []string `json:"requestedScopes"`
	RootCA          string   `json:"rootCA"`
}

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
	dnsSubDomain, err := getDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("Component ArgoCD failed retrieving DNS sub domain: %v", err)
		return err
	}

	keycloakHost := "keycloak." + dnsSubDomain
	//argocdHost := "argocd." + dnsSubDomain
	keycloakUrl := fmt.Sprintf("https://%s/%s", keycloakHost, "auth/realms/verrazzano-system")

	ctx.Log().Debugf("Getting ArgoCD TLS root CA")
	caCert, err := GetRootCA(ctx)
	if err != nil {
		ctx.Log().Errorf("Failed to get ArgoCD TLS root CA: %v", err)
		return err
	}

	/*t, err := template.New("oidcconfig_template").Parse(oidcConfigTemplate)
	if err != nil {
		return err
	}*/

	conf := &OIDCConfig{
		Name:         "Keycloak",
		Issuer:       keycloakUrl,
		ClientID:     "argocd",
		ClientSecret: "$oidc.keycloak.clientSecret",
		RequestedScopes: []string{
			"openid",
			"profile",
			"email",
			"groups",
		},
		RootCA: string(caCert),
	}

	ctx.Log().Infof("cacert - %s", string(caCert))
	keycloakUrl = fmt.Sprintf("https://%s/%s", keycloakHost, "auth/realms/verrazzano-system")
	//b := &bytes.Buffer{}
	data, err := yaml.Marshal(conf)
	if err != nil {
		fmt.Println(err)
		return err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-cm",
			Namespace: constants.ArgoCDNamespace,
		},
		Data: map[string]string{
			"oidc.config": string(data),
		},
	}

	// Add the oidc configuration to enable our keycloak authentication.
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), cm, func() error {
		return nil
	}); err != nil {
		return err
	}

	ctx.Log().Debugf("patchArgoCDConfigMap: ArgoCD cm operation result: %v", err)
	return err
}

// patchArgoCDRbacConfigMap
func patchArgoCDRbacConfigMap(ctx spi.ComponentContext) error {
	rbaccm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-rbac-cm",
			Namespace: constants.ArgoCDNamespace,
		},
	}

	var policyString = `g, verrazzano-admins, role:admin`
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

// GetRootCA gets the root CA certificate from the argocd TLS secret. If the secret does not exist, we
// return a nil slice.
func GetRootCA(ctx spi.ComponentContext) ([]byte, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.ArgoCDNamespace,
		Name:      common.ArgoCDIngressCAName}

	if err := ctx.Client().Get(context.TODO(), nsName, secret); err != nil {
		return nil, err
	}

	return secret.Data[common.ArgoCDCACert], nil
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

// getDNSDomain returns the DNS Domain
func getDNSDomain(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	dnsDomain := fmt.Sprintf("%s.%s", vz.Spec.EnvironmentName, dnsSuffix)
	return dnsDomain, nil
}
