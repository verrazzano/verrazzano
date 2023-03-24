// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

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
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// ValidateCertDate validates the certs.
func (r *CertificateRotationManagerReconciler) ValidateCertDate(certContent []byte) (bool, error) {
	block, _ := pem.Decode(certContent)
	if block == nil {
		return false, r.log.ErrorfNewErr("unable to parse the certificate")
	}
	certs, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false, r.log.ErrorfNewErr(err.Error())
	}
	deadline := time.Now().Add(time.Hour * r.CompareWindow)
	if deadline.After(certs.NotAfter) {
		r.log.Progressf("certificate for %s expires too soon: %s less than %s away",
			certs.Subject.CommonName,
			certs.NotAfter.Format(time.RFC3339),
			r.CompareWindow)
		return true, nil

	}
	r.log.Progressf("certificate for %s has validity %s",
		certs.Subject.CommonName,
		certs.NotAfter.Sub(deadline).Round(time.Hour))
	return false, nil
}

// GetSecretData checks if secret exists in namespace or not
// if exists then return secret data else return nil
func (r *CertificateRotationManagerReconciler) GetSecretData(secretName string) (corev1.Secret, []byte) {
	sec := corev1.Secret{}
	err := r.Get(context.TODO(), clipkg.ObjectKey{
		Namespace: r.WatchNamespace,
		Name:      secretName,
	}, &sec)
	if err != nil && apierrors.IsNotFound(err) {
		r.log.Errorf("an error while listing the certificate secret %v", secretName)
		return sec, nil
	}
	r.log.Debugf("successfully found the certificate secret %v", secretName)
	if val, ok := sec.Data["tls.crt"]; ok {
		return sec, val
	}
	return sec, nil
}

// DeleteSecret deletes the tls secret.
func (r *CertificateRotationManagerReconciler) DeleteSecret(secretName corev1.Secret) error {
	if err := r.Delete(context.TODO(), &secretName, &clipkg.DeleteOptions{}); err != nil {
		r.log.Errorf("an error while deleting the certificate secret %v in namespace %v with error %v", secretName.Name, r.WatchNamespace, err)
		return r.log.ErrorfNewErr("an while deleting the secret")
	}
	r.log.Infof("successfully deleted the secret %v in namespace %v ", secretName.Name, r.WatchNamespace)
	return nil
}

func (r *CertificateRotationManagerReconciler) RolloutRestartDeployment() error {
	deployment := appsv1.Deployment{}
	var err error
	err = r.Get(context.TODO(), clipkg.ObjectKey{Namespace: r.TargetNamespace, Name: r.TargetDeployment}, &deployment)
	r.log.Debug("deployment listed", deployment.Name)
	if err != nil && apierrors.IsNotFound(err) {
		r.log.Errorf("an error while listing the deployment in namespace %v", r.TargetNamespace)
		return err
	}
	//deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(generation)
	//if _, err := goClient.AppsV1().Deployments(deployment.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{}); err != nil {
	//	return log.ErrorfNewErr("Failed, error updating Deployment %s annotation to force a pod restart", deployment.Name)
	//}
	err = r.Patch(context.TODO(), &deployment, clipkg.RawPatch(types.StrategicMergePatchType, generatePatch()))
	if err != nil {
		r.log.Errorf("an error while patching the deployment %v in namespace %v", r.TargetDeployment, r.TargetNamespace)
		return err
	}
	r.log.Infof("successfully restart the deployment %v in namespace %v ", r.TargetDeployment, r.WatchNamespace)
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
