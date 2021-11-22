// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
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
			Name: CattleSystem,
		},
	}
	log.Debugf("Creating %s namespace", CattleSystem)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, namespace, func() error {
		log.Debugf("Ensuring %s label is present on %s namespace", namespaceLabelKey, CattleSystem)
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		namespace.Labels[namespaceLabelKey] = Name
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
		rancherCaSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: CattleSystem,
				Name:      rancherTLSSecretName,
			},
		}
		log.Infof("Copying default Verrazzano secret to Rancher namespace")
		if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, rancherCaSecret, func() error {
			rancherCaSecret.Data = map[string][]byte{
				"cacerts.pem": defaultSecret.Data["ca.cert"],
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

func createAdditionalCertificates(log *zap.SugaredLogger, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if (cm != nil && cm.Certificate.Acme != vzapi.Acme{} && useAdditionalCAs(cm.Certificate.Acme)) {
		log.Infof("Creating additional Rancher certificates for non-production environment")
		script := filepath.Join(config.GetInstallDir(), "install-rancher-certificates.sh")
		if _, stderr, err := bashFunc(script); err != nil {
			log.Errorf("Rancher pre install: Failed to install letsEncrypt certificates: %s: %s", err, stderr)
			return err
		}
	}
	return nil
}
