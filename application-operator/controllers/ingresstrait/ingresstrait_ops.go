// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package ingresstrait

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	certapiv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Cleanup cleans up the generated certificates and secrets associated with the given app config
func Cleanup(appName types.NamespacedName, client client.Client, log logr.Logger) (err error) {
	certName, err := buildCertificateNameFromAppName(appName)
	if err != nil {
		log.Error(err, "Error building certificate name", "appName", appName.Name)
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
func cleanupCert(certName string, c client.Client, log logr.Logger) (err error) {
	nsn := types.NamespacedName{Name: certName, Namespace: constants.IstioSystemNamespace}
	cert := &certapiv1alpha2.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		},
	}
	// Delete the cert, ignore not found
	log.Info("Deleting cert", "cert", nsn)
	err = c.Delete(context.TODO(), cert, &client.DeleteOptions{})
	if err != nil {
		// integration tests do not install cert manager so no match error is generated
		if k8serrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			log.Info("NotFound deleting cert", "cert", nsn)
			return nil
		}
		log.Error(err, "Error deleting the cert", "cert", nsn)
		return err
	}
	log.Info("Ingress certificate deleted", "cert", nsn)
	return nil
}

// cleanupSecret deletes up the generated secret for the given app config
func cleanupSecret(certName string, c client.Client, log logr.Logger) (err error) {
	secretName := fmt.Sprintf("%s-secret", certName)
	nsn := types.NamespacedName{Name: secretName, Namespace: constants.IstioSystemNamespace}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: nsn.Namespace,
			Name:      nsn.Name,
		},
	}
	// Delete the secret, ignore not found
	log.Info("Deleting secret", "secret", nsn)
	err = c.Delete(context.TODO(), secret, &client.DeleteOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("NotFound deleting secret", "secret", nsn)
			return nil
		}
		log.Error(err, "Error deleting the secret", "secret", nsn)
		return err
	}
	log.Info("Ingress secret deleted", "cert", nsn)
	return nil
}

// buildCertificateNameFromAppName will construct a cert name from the app name.
func buildCertificateNameFromAppName(appName types.NamespacedName) (string, error) {
	if len(appName.Name) == 0 {
		return "", errors.New("OAM app name label missing from metadata, unable to generate certificate name")

	}
	return fmt.Sprintf("%s-%s-cert", appName.Namespace, appName.Name), nil
}
