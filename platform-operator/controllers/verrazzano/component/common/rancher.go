// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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

// GetRancherMgmtAPIGVRForResource returns a management.cattle.io/v3 GroupVersionKind structure for specified kind
func GetRancherMgmtAPIGVRForResource(kind string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    APIGroupRancherManagement,
		Version:  APIGroupVersionRancherManagement,
		Resource: "kontainerdrivers",
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

// Retry executes the provided function repeatedly, retrying until the function
// returns done = true, or exceeds the given timeout.
// errors will be logged, but will not trigger retry to stop unless retryOnError is false
func Retry(backoff wait.Backoff, log vzlog.VerrazzanoLogger, retryOnError bool, fn wait.ConditionFunc) error {
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		done, err := fn()
		lastErr = err
		if err != nil && retryOnError {
			log.Infof("Retrying after error: %v", err)
			return done, nil
		}
		return done, err
	})
	if err == wait.ErrWaitTimeout {
		if lastErr != nil {
			err = lastErr
		}
	}
	return err
}

// ActivateKontainerDriver - Create or update the kontainerdrivers.management.cattle.io object that
// registers the ociocne driver
func ActivateKontainerDriver(ctx spi.ComponentContext) error {
	kontainerDriverObjectName := "ociocneengine"
	// Nothing to do if Capi is not enabled
	if !vzcr.IsClusterAPIEnabled(ctx.EffectiveCR()) {
		return nil
	}

	// Setup dynamic client
	dynClient, err := k8sutil.GetDynamicClient()
	if err != nil {
		return fmt.Errorf("Failed to get dynamic client: %v", err)
	}

	// Get the driver object
	var driverObj *unstructured.Unstructured
	gvr := GetRancherMgmtAPIGVRForResource("kontainerdrivers")
	driverObj, err = dynClient.Resource(gvr).Get(context.TODO(), kontainerDriverObjectName, metav1.GetOptions{})
	if err != nil {
		// Keep trying until the resource is found
		return fmt.Errorf("Failed to get %s/%s/%s %s: %v", gvr.Resource, gvr.Group, gvr.Version, kontainerDriverObjectName, err)
	}

	// Activate the driver
	driverObj.UnstructuredContent()["spec"].(map[string]interface{})["active"] = true
	_, err = dynClient.Resource(gvr).Update(context.TODO(), driverObj, metav1.UpdateOptions{})
	return err
}
