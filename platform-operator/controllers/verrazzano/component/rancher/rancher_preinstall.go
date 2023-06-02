// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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
		namespace.Labels[constants.VerrazzanoManagedKey] = common.RancherName
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// copyDefaultCACertificate copies the defaultVerrazzanoName TLS Secret to the ComponentNamespace for use by Rancher
// This method will only copy defaultVerrazzanoName if default CA certificates are being used.
func copyDefaultCACertificate(log vzlog.VerrazzanoLogger, c client.Client, vz *vzapi.Verrazzano) error {
	clusterIssuer := vz.Spec.Components.ClusterIssuer
	if clusterIssuer == nil {
		// Not necessarily an error, since CM and the ClusterIssuer could be disabled
		log.Progressf("No cluster issuer found, skipping CA certificate bundle configuration")
		return nil
	}
	isCAIssuer, err := clusterIssuer.IsCAIssuer()
	if err != nil {
		return err
	}
	if isCAIssuer {
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
			return err
		}
		if len(caSecret.Data[certKey]) < 1 {
			return log.ErrorfNewErr("Failed, secret %s/%s does not have a value for %s",
				caSecretNamespace,
				caSecretName, certKey)
		}
		rancherCaSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: common.CattleSystem,
				Name:      rancherTLSSecretName,
			},
		}
		log.Debugf("Copying default Verrazzano secret to Rancher namespace")
		if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, rancherCaSecret, func() error {
			rancherCaSecret.Data = map[string][]byte{
				caCertsPem: caSecret.Data[certKey],
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func isUsingDefaultCACertificate(cm *vzapi.ClusterIssuerComponent) bool {
	if cm == nil {
		return false
	}
	isDefaultCA, _ := cm.IsDefaultIssuer()
	return isDefaultCA
}
