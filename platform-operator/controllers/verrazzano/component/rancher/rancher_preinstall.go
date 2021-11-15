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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createCattleSystemNamespaceIfNotExists(log *zap.SugaredLogger, c client.Client) error {
	namespacedName := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      ComponentNamespace,
	}
	namespace := &v1.Namespace{}
	err := c.Get(context.TODO(), namespacedName, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) { // if the namespace does not exist, create it
			log.Debugf("Creating %v namespace for Rancher", ComponentNamespace)
			return c.Create(context.TODO(), &v1.Namespace{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name: ComponentNamespace,
					Labels: map[string]string{
						NamespaceLabelKey: ComponentNamespace,
					},
				},
			})
		}
		log.Errorf("Failed to create %v namespace", ComponentNamespace)
		return err
	} else { // namespace exists, and needs to be labelled
		namespaceMerge := client.MergeFrom(namespace.DeepCopy())
		namespace.Labels[NamespaceLabelKey] = ComponentName
		log.Debugf("Patching %v namespace", ComponentNamespace)
		return c.Patch(context.TODO(), namespace, namespaceMerge)
	}
}

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
				Name:      RancherTlsSecret,
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
	if (cm != nil && cm.Certificate.Acme != vzapi.Acme{} && cm.Certificate.Acme.Environment != "production") {
		log.Infof("Creating additional Rancher certificates for non-production environment")
		script := filepath.Join(config.GetInstallDir(), "install-rancher-certificates.sh")
		if _, stderr, err := os2.RunBash(script); err != nil {
			log.Errorf("Rancher pre install: Failed to install letsEncrypt certificates: %s: %s", err, stderr)
			return err
		}
	}
	return nil
}
