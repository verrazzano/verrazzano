// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/certs"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	spi "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzresource "github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// additionalTLS is an optional tls secret that contains additional CA
const additionalTLS = "tls-ca-additional"

// createCattleSystemNamespace creates the cattle-system namespace if it does not exist
func createCattleSystemNamespace(log vzlog.VerrazzanoLogger, c client.Client) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.CattleSystem,
		},
	}
	log.Debugf("Creating %s namespace", common.CattleSystem)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, namespace, func() error {
		log.Debugf("Ensuring %s label is present on %s namespace", constants.VerrazzanoManagedKey, common.CattleSystem)
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		namespace.Labels[constants.VerrazzanoManagedKey] = common.CattleSystem
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// copyPrivateCABundles detects if a private CA bundle is in use (default, customer CA, or Let's Encrypt staging)
// and sets up the "cattle-system/tls-ca" secret with the corresponding CA bundle data for Rancher to be able to use to validate
// client certificates issued by those CAs.
//
// If private CAs are not in use, we will clean up the cattle-system/tls-ca secret if it exists
func copyPrivateCABundles(log vzlog.VerrazzanoLogger, c client.Client, vz *vzapi.Verrazzano) error {
	log.Debugf("Cleaning up legacy Rancher additional CA secret %s/%s if it exists", common.CattleSystem, additionalTLS)
	err := vzresource.Resource{
		Namespace: common.CattleSystem,
		Name:      additionalTLS,
		Client:    c,
		Object:    &v1.Secret{},
		Log:       log,
	}.Delete()
	if err != nil {
		return err
	}

	if isPrivateIssuer, _ := certs.IsPrivateIssuer(vz.Spec.Components.ClusterIssuer); !isPrivateIssuer {
		// If we drop through to this point we are not using a private CA bundle and should clean up the secret
		log.Debugf("Private CA bundle not in use, cleaning up Rancher private CA secret %s", rancherTLSCASecretName)
		return vzresource.Resource{
			Namespace: common.CattleSystem,
			Name:      rancherTLSCASecretName,
			Client:    c,
			Object:    &v1.Secret{},
			Log:       log,
		}.Delete()
	}

	privateCASecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconst.VerrazzanoSystemNamespace,
			Name:      vzconst.PrivateCABundle,
		},
	}
	if err := c.Get(context.TODO(), client.ObjectKeyFromObject(privateCASecret), privateCASecret); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		log.Progressf("Private CA bundle secret not found, retrying...")
		return spi.RetryableError{Source: ComponentName, Cause: err}
	}

	bundleData, found := privateCASecret.Data[vzconst.CABundleKey]
	if !found {
		return log.ErrorfThrottledNewErr("Private CA secret %s exists but expected bundle not found", privateCASecret.Name)
	}

	rancherCaSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      rancherTLSCASecretName,
		},
	}
	// If there is private CA bundle data, create/update the tls-ca secret
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, rancherCaSecret, func() error {
		rancherCaSecret.Data = map[string][]byte{
			caCertsPem: bundleData,
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// isUsingDefaultCACertificate Returns true if the default self-signed issuer is in use
func isUsingDefaultCACertificate(cm *vzapi.ClusterIssuerComponent) bool {
	if cm == nil {
		return false
	}
	isDefaultCA, _ := cm.IsDefaultIssuer()
	return isDefaultCA
}
