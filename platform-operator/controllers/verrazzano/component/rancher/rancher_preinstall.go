// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	vzresource "github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/vzcr"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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
	// Determine if there is any private CA bundles in use and get the bundle data
	bundleData, err := getPrivateBundleData(log, c, vz)
	if err != nil {
		return err
	}

	rancherCaSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.CattleSystem,
			Name:      rancherTLSCASecretName,
		},
	}

	// If there is private CA bundle data, create/update the tls-ca secret
	if len(bundleData) > 0 {
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

	// If we drop through to this point we are not using a private CA bundle and should clean up the secret
	log.Debugf("Private CA bundle not in use, cleaning up Rancher private CA secret %s/%s")
	return vzresource.Resource{
		Name:      rancherCaSecret.Name,
		Namespace: rancherCaSecret.Namespace,
		Client:    c,
		Object:    &v1.Secret{},
		Log:       log,
	}.Delete()
}

// getPrivateBundleData returns the CA cert bundle when a private CA configuration is in use (self-signed, customer CA,
// or Let's Encrypt Staging); this data will be stored in the tls-ca secret used by Rancher for private/untrusted CA
// configurations
func getPrivateBundleData(log vzlog.VerrazzanoLogger, c client.Client, vz *vzapi.Verrazzano) ([]byte, error) {
	clusterIssuer := vz.Spec.Components.ClusterIssuer
	if clusterIssuer == nil {
		// Not necessarily an error, since CM and the ClusterIssuer could be disabled
		log.Progressf("No cluster issuer found, skipping CA certificate bundle configuration")
		return []byte{}, nil
	}

	isCAIssuer, err := clusterIssuer.IsCAIssuer()
	if err != nil {
		return []byte{}, err
	}
	var isLetsEncryptStagingEnv bool
	if clusterIssuer.LetsEncrypt != nil {
		isLetsEncryptStagingEnv = vzcr.IsLetsEncryptStagingEnv(*clusterIssuer.LetsEncrypt)
	}

	var bundleData []byte
	if isCAIssuer {
		log.Infof("Getting private CA bundle for Rancher")
		if bundleData, err = createPrivateCABundle(log, c, clusterIssuer); err != nil {
			return []byte{}, err
		}
	} else if isLetsEncryptStagingEnv {
		log.Infof("Getting Let's Encrypt Staging CA bundle for Rancher")
		if bundleData, err = createLetsEncryptStagingBundle(); err != nil {
			return []byte{}, err
		}
	}
	return bundleData, nil
}

// createPrivateCABundle Obtains the private CA bundle for the self-signed/customer-provided CA configuration
func createPrivateCABundle(log vzlog.VerrazzanoLogger, c client.Client, clusterIssuer *vzapi.ClusterIssuerComponent) ([]byte, error) {
	caSecretNamespace := clusterIssuer.ClusterResourceNamespace
	caSecretName := clusterIssuer.CA.SecretName
	namespacedName := types.NamespacedName{
		Namespace: caSecretNamespace,
		Name:      caSecretName,
	}
	certKey := caCert
	if isDefault, _ := clusterIssuer.IsDefaultIssuer(); !isDefault {
		certKey = customCACertKey
	}
	caSecret := &v1.Secret{}
	if err := c.Get(context.TODO(), namespacedName, caSecret); err != nil {
		return []byte{}, err
	}
	if len(caSecret.Data[certKey]) < 1 {
		return nil, log.ErrorfNewErr("Failed, secret %s/%s does not have a value for %s",
			caSecretNamespace,
			caSecretName, certKey)
	}
	return caSecret.Data[certKey], nil
}

// isUsingDefaultCACertificate Returns true if the default self-signed issuer is in use
func isUsingDefaultCACertificate(cm *vzapi.ClusterIssuerComponent) bool {
	if cm == nil {
		return false
	}
	isDefaultCA, _ := cm.IsDefaultIssuer()
	return isDefaultCA
}
