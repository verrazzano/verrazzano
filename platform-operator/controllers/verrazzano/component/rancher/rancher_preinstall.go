// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createRancherOperatorNamespace(log *zap.SugaredLogger, c client.Client) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: OperatorNamespace,
		},
	}
	log.Debugf("Creating %s namespace", OperatorNamespace)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, namespace, func() error {
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// createCattleSystemNamespace creates the cattle-system namespace if it does not exist
func createCattleSystemNamespace(log *zap.SugaredLogger, c client.Client) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.CattleSystem,
		},
	}
	log.Debugf("Creating %s namespace", common.CattleSystem)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, namespace, func() error {
		log.Debugf("Ensuring %s label is present on %s namespace", namespaceLabelKey, common.CattleSystem)
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		namespace.Labels[namespaceLabelKey] = common.RancherName
		return nil
	}); err != nil {
		return err
	}

	return nil
}

//copyDefaultCACertificate copies the defaultVerrazzanoName TLS Secret to the ComponentNamespace for use by Rancher
//This method will only copy defaultVerrazzanoName if default CA certificates are being used.
func copyDefaultCACertificate(log *zap.SugaredLogger, c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if isUsingDefaultCACertificate(cm) {
		namespacedName := types.NamespacedName{Namespace: defaultSecretNamespace, Name: defaultVerrazzanoName}
		defaultSecret := &v1.Secret{}
		if err := c.Get(context.TODO(), namespacedName, defaultSecret); err != nil {
			return err
		}
		if len(defaultSecret.Data[caCert]) < 1 {
			return fmt.Errorf("%s/%s does not have a value for %s", defaultSecretNamespace, defaultVerrazzanoName, caCert)
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
				caCertsPem: defaultSecret.Data[caCert],
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

func isUsingDefaultCACertificate(cm *vzapi.CertManagerComponent) bool {
	return cm != nil &&
		cm.Certificate.CA != vzapi.CA{} &&
		cm.Certificate.CA.SecretName == defaultVerrazzanoName &&
		cm.Certificate.CA.ClusterResourceNamespace == defaultSecretNamespace
}

func createAdditionalCertificates(log *zap.SugaredLogger, c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if (cm != nil && cm.Certificate.Acme != vzapi.Acme{} && useAdditionalCAs(cm.Certificate.Acme)) {
		log.Debugf("Creating additional Rancher certificates for non-production environment")
		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: common.CattleSystem,
				Name:      common.RancherAdditionalIngressCAName,
			},
		}

		if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, secret, func() error {
			builder := &certBuilder{
				hc: &http.Client{},
			}
			if err := builder.buildLetsEncryptStagingChain(); err != nil {
				return err
			}
			secret.Data = map[string][]byte{
				common.RancherCAAdditionalPem: builder.cert,
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
