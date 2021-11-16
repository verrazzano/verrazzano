// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	os2 "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createCattleSystemNamespaceIfNotExists(log *zap.SugaredLogger, c client.Client) error {
	namespace := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: ComponentNamespace,
			Labels: map[string]string{
				namespaceLabelKey: ComponentNamespace,
			},
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, namespace, func() error {
		namespace.Labels[namespaceLabelKey] = ComponentName
		return nil
	}); err != nil {
		log.Errorf("Failed to create %v namespace", ComponentNamespace)
		return err
	}

	return nil
}

//copyDefaultCACertificate copies the defaultVerrazzanoSecretName TLS Secret to the ComponentNamespace for use by Rancher
//This method will only copy defaultVerrazzanoSecretName if default CA certificates are being used.
func copyDefaultCACertificate(log *zap.SugaredLogger, c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if isUsingDefaultCACertificate(cm) {
		log.Infof("Copying default Verrazzano secret to Rancher namespace")
		namespacedName := types.NamespacedName{Namespace: defaultSecretNamespace, Name: defaultVerrazzanoSecretName}
		defaultSecret := &v1.Secret{}
		if err := c.Get(context.TODO(), namespacedName, defaultSecret); err != nil {
			return err
		}
		secretData := map[string][]byte{
			"cacerts.pem": defaultSecret.Data["ca.cert"],
		}
		rancherCaSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ComponentNamespace,
				Name:      rancherTLSSecretName,
			},
			Data: secretData,
		}
		_, err := controllerruntime.CreateOrUpdate(context.TODO(), c, rancherCaSecret, func() error {
			rancherCaSecret.Data = secretData
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func isUsingDefaultCACertificate(cm *vzapi.CertManagerComponent) bool {
	return cm != nil &&
		cm.Certificate.CA != vzapi.CA{} &&
		cm.Certificate.CA.SecretName == defaultVerrazzanoSecretName &&
		cm.Certificate.CA.ClusterResourceNamespace == defaultSecretNamespace
}

func createAdditionalCertificates(log *zap.SugaredLogger, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if (cm != nil && cm.Certificate.Acme != vzapi.Acme{} && useAdditionalCAs(cm.Certificate.Acme)) {
		log.Infof("Creating additional Rancher certificates for non-production environment")
		script := filepath.Join(config.GetInstallDir(), "install-rancher-certificates.sh")
		if _, stderr, err := os2.RunBash(script); err != nil {
			log.Errorf("Rancher pre install: Failed to install letsEncrypt certificates: %s: %s", err, stderr)
			return err
		}
	}
	return nil
}
