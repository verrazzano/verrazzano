// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"crypto/x509"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Rancher HTTPS Configuration
const (
	// RancherName is the name of the component
	RancherName = "rancher"
	// CattleSystem is the namespace of the component
	CattleSystem                            = "cattle-system"
	RancherIngressCAName                    = "tls-rancher-ingress"
	RancherAdminSecret                      = "rancher-admin-secret"
	RancherCACert                           = "ca.crt"
	AuthConfigKeycloakAttributeClientSecret = "clientSecret"
	APIGroupRancherManagement               = "management.cattle.io"
	APIGroupVersionRancherManagement        = "v3"
	AuthConfigKeycloak                      = "keycloakoidc"
	SettingFirstLogin                       = "first-login"
)

var GVKAuthConfig = GetRancherMgmtAPIGVKForKind("AuthConfig")
var GVKSetting = GetRancherMgmtAPIGVKForKind("Setting")

// GetAdminSecret fetches the Rancher admin secret
func GetAdminSecret(c client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      RancherAdminSecret}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// GetRootCA gets the root CA certificate from the Rancher TLS secret. If the secret does not exist, we
// return a nil slice.
func GetRootCA(c client.Reader) ([]byte, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      RancherIngressCAName}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return secret.Data[RancherCACert], nil
}

// GetAdditionalCA fetches the Rancher additional CA secret
// returns empty byte array of the secret tls-ca-additional is not found
func GetAdditionalCA(c client.Reader) []byte {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      constants.AdditionalTLS}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return []byte{}
	}

	return secret.Data[constants.AdditionalTLSCAKey]
}

func CertPool(certs ...[]byte) *x509.CertPool {
	certPool := x509.NewCertPool()
	for _, cert := range certs {
		if len(cert) > 0 {
			certPool.AppendCertsFromPEM(cert)
		}
	}
	return certPool
}

// GetRancherMgmtAPIGVKForKind returns a management.cattle.io/v3 GroupVersionKind structure for specified kind
func GetRancherMgmtAPIGVKForKind(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   APIGroupRancherManagement,
		Version: APIGroupVersionRancherManagement,
		Kind:    kind,
	}
}

// UpdateKeycloakOIDCAuthConfig updates the keycloakoidc AuthConfig CR with specified attributes
func UpdateKeycloakOIDCAuthConfig(ctx spi.ComponentContext, attributes map[string]interface{}) error {
	log := ctx.Log()
	c := ctx.Client()
	keycloakAuthConfig := unstructured.Unstructured{}
	keycloakAuthConfig.SetGroupVersionKind(GVKAuthConfig)
	keycloakAuthConfigName := types.NamespacedName{Name: AuthConfigKeycloak}
	err := c.Get(context.Background(), keycloakAuthConfigName, &keycloakAuthConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Progressf("Rancher component is waiting for Keycloak authConfig to exist")
			return ctrlerrors.RetryableError{}
		}
		return log.ErrorfThrottledNewErr("Failed to fetch Keycloak authConfig: %v", err.Error())

	}

	authConfig := keycloakAuthConfig.UnstructuredContent()
	for key, value := range attributes {
		authConfig[key] = value
	}
	err = c.Update(context.Background(), &keycloakAuthConfig, &client.UpdateOptions{})
	if err != nil {
		return log.ErrorfThrottledNewErr("failed configuring Keycloak as OIDC provider for rancher: %s", err.Error())
	}

	return nil
}
