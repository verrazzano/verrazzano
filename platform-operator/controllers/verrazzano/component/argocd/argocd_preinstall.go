// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isUsingDefaultCACertificate(cm *vzapi.CertManagerComponent) bool {
	return cm != nil &&
		cm.Certificate.CA != vzapi.CA{} &&
		cm.Certificate.CA.SecretName == defaultVerrazzanoName &&
		cm.Certificate.CA.ClusterResourceNamespace == defaultSecretNamespace
}

// copyDefaultCACertificate copies the defaultVerrazzanoName TLS Secret to the ComponentNamespace for use by Argo CD
// This method will only copy defaultVerrazzanoName if default CA certificates are being used.
func copyDefaultCACertificate(log vzlog.VerrazzanoLogger, c client.Client, vz *vzapi.Verrazzano) error {
	cm := vz.Spec.Components.CertManager
	if isUsingDefaultCACertificate(cm) {
		namespacedName := types.NamespacedName{Namespace: defaultSecretNamespace, Name: defaultVerrazzanoName}
		defaultSecret := &v1.Secret{}
		if err := c.Get(context.TODO(), namespacedName, defaultSecret); err != nil {
			return err
		}
		if len(defaultSecret.Data[caCert]) < 1 {
			return log.ErrorfNewErr("Failed, secret %s/%s does not have a value for %s", defaultSecretNamespace, defaultVerrazzanoName, caCert)
		}
		argoCDCaSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: common.ArgoCDNamespace,
				Name:      argoCDTLSSecretName,
			},
		}
		log.Debugf("Copying default Verrazzano secret to ArgoCD namespace")
		if _, err := controllerruntime.CreateOrUpdate(context.TODO(), c, argoCDCaSecret, func() error {
			argoCDCaSecret.Data = map[string][]byte{
				caCertsPem: defaultSecret.Data[caCert],
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}
