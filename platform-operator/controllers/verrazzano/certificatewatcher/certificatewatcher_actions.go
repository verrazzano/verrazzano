// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificatewatcher

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"log"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// ValidateCertDate validates the certs.
func (sw *CertificateWatcher) ValidateCertDate(certContent []byte) (bool, error) {
	block, _ := pem.Decode(certContent)
	if block == nil {
		log.Fatal("failed to parse PEM block containing the public key")
		return false, fmt.Errorf("unable to parse the certificate")
	}
	certs, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, fmt.Errorf(err.Error())
	}
	deadline := time.Now().Add(expiry)
	if deadline.After(certs.NotAfter) {
		sw.log.Infof("cert for %s expires too soon: %s less than %s away",
			certs.Subject.CommonName,
			certs.NotAfter.Format(time.RFC3339),
			expiry)
		return true, nil

	}
	sw.log.Infof("cert for %s has validity %s",
		certs.Subject.CommonName,
		certs.NotAfter.Sub(deadline).Round(time.Hour))
	return false, nil
}

// GetSecretData checks if secret exists in namespace or not
// if exists then return secret data else return nil
func (sw *CertificateWatcher) GetSecretData(secretName string) (corev1.Secret, []byte) {
	sec := corev1.Secret{}
	err := sw.client.Get(context.TODO(), clipkg.ObjectKey{
		Namespace: sw.watchNamespace,
		Name:      secretName,
	}, &sec)
	if err != nil && apierrors.IsNotFound(err) {
		sw.log.Errorf("an error while listing the certificate secret %v", secretName)
		return sec, nil
	}
	sw.log.Debugf("successfully found the certificate secret %v", secretName)
	if val, ok := sec.Data["tls.crt"]; ok {
		return sec, val
	}
	return sec, nil
}

// DeleteSecret deletes the tls secret.
func (sw *CertificateWatcher) DeleteSecret(secretName corev1.Secret) error {
	if err := sw.client.Delete(context.TODO(), &secretName, &clipkg.DeleteOptions{}); err != nil {
		sw.log.Errorf("an error while deleting the certificate secret %v in namespace %v with error %v", secretName.Name, sw.watchNamespace, err)
		return fmt.Errorf("an while deleting the secret")
	}
	sw.log.Infof("successfully deleted the secret %v in namespace %v ", secretName.Name, sw.watchNamespace)
	return nil
}

func (sw *CertificateWatcher) RolloutRestartDeployment() error {
	deployment := appsv1.Deployment{}
	var err error
	err = sw.client.Get(context.TODO(), clipkg.ObjectKey{Namespace: sw.targetNamespace, Name: sw.targetDeployment}, &deployment)
	sw.log.Debug("deployment listed", deployment.Name)
	if err != nil && apierrors.IsNotFound(err) {
		sw.log.Errorf("an error while listing the deployment in namespace %v", sw.targetNamespace)
		return err
	}
	err = sw.client.Patch(context.TODO(), &deployment, clipkg.RawPatch(types.StrategicMergePatchType, generatePatch()))
	sw.log.Infof("successfully restart the deployment %v in namespace %v ", sw.targetDeployment, sw.watchNamespace)
	if err != nil {
		sw.log.Errorf("an error while patching the deployment %v in namespace %v", sw.targetDeployment, sw.targetNamespace)
		return err
	}
	return nil
}

// generatePatch returns patch data used to restart the deployment.
func generatePatch() []byte {
	patchTime := time.Now().Format(time.RFC3339)
	mergePatch, err := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"annotations": map[string]interface{}{
						"kubectl.kubernetes.io/restartedAt": patchTime,
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Println("error while doing operation")
	}
	return mergePatch
}
