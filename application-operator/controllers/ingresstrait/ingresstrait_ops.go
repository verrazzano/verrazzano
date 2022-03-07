// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ingresstrait

import (
	"context"
	"errors"
	"fmt"

	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Cleanup cleans up the generated certificates and secrets associated with the given app config
func Cleanup(appName types.NamespacedName, client client.Client, log vzlog.VerrazzanoLogger) (err error) {
	certName, err := buildCertificateNameFromAppName(appName)
	if err != nil {
		log.Errorf("Failed building certificate name: %s", err)
		return err
	}
	err = cleanupCert(certName, client, log)
	if err != nil {
		return
	}
	err = cleanupSecret(certName, client, log)
	if err != nil {
		return
	}
	return
}

// cleanupCert deletes up the generated certificate for the given app config
func cleanupCert(certName string, c client.Client, log vzlog.VerrazzanoLogger) (err error) {
	nsn := types.NamespacedName{Name: certName, Namespace: constants.IstioSystemNamespace}
	cert := &certapiv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		},
	}
	// Delete the cert, ignore not found
	log.Debugf("Deleting cert: %s", nsn.Name)
	err = c.Delete(context.TODO(), cert, &client.DeleteOptions{})
	if err != nil {
		// integration tests do not install cert manager so no match error is generated
		if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			log.Debugf("NotFound deleting cert %s", nsn.Name)
			return nil
		}
		log.Errorf("Failed deleting the cert %s", nsn.Name)
		return err
	}
	log.Debugf("Ingress certificate %s deleted", nsn.Name)
	return nil
}

// cleanupSecret deletes up the generated secret for the given app config
func cleanupSecret(certName string, c client.Client, log vzlog.VerrazzanoLogger) (err error) {
	secretName := fmt.Sprintf("%s-secret", certName)
	nsn := types.NamespacedName{Name: secretName, Namespace: constants.IstioSystemNamespace}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		},
	}
	// Delete the secret, ignore not found
	log.Debugf("Deleting secret %s", nsn.Name)
	err = c.Delete(context.TODO(), secret, &client.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Debugf("NotFound deleting secret %s", nsn)
			return nil
		}
		log.Errorf("Failed deleting the secret %s: %v", nsn.Name, err)
		return err
	}
	log.Debugf("Ingress secret %s deleted", nsn.Name)
	return nil
}

// buildCertificateNameFromAppName will construct a cert name from the app name.
func buildCertificateNameFromAppName(appName types.NamespacedName) (string, error) {
	if len(appName.Name) == 0 {
		return "", errors.New("OAM app name label missing from metadata, unable to generate certificate name")

	}
	return fmt.Sprintf("%s-%s-cert", appName.Namespace, appName.Name), nil
}
