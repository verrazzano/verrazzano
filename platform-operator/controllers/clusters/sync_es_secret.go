// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	k8net "k8s.io/api/networking/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	caKey       = "ca.crt"
	passwordKey = "password"
	usernameKey = "username"
	urlKey      = "url"
	vmiIngest   = "vmi-system-es-ingest"
)

// Create a Elasticsearch secret that has the fields needed to send logs from the
// managed cluster to Elasticsearch running in the admin cluster.
func (r *VerrazzanoManagedClusterReconciler) syncElasticsearchSecret(vmc *clusterapi.VerrazzanoManagedCluster) error {
	// Get the info needed to build the elasicsearch secret
	url, err := r.getElasticsearchURL()
	if err != nil {
		return err
	}
	vzSecret, err := r.getVzSecret()
	if err != nil {
		return err
	}
	tlsSecret, err := r.getTLSSecret()
	if err != nil {
		return err
	}

	// Build the secret data
	secretData := make(map[string][]byte)
	secretData[caKey] = tlsSecret.Data[caKey]
	secretData[usernameKey] = vzSecret.Data[usernameKey]
	secretData[passwordKey] = vzSecret.Data[passwordKey]
	secretData[urlKey] = []byte(url)

	// Create/update the Elasticsearch secret
	_, err = r.createOrUpdateElasticsearchSecret(vmc, secretData)
	if err != nil {
		return err
	}

	return nil
}

// Create or update the Elasticsearch secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateElasticsearchSecret(vmc *clusterapi.VerrazzanoManagedCluster, secretData map[string][]byte) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = constants.VerrazzanoMultiClusterNamespace
	secret.Name = GetElasticsearchSecretName(vmc.Name)

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateElasticsearchSecret(&secret, secretData)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret
func (r *VerrazzanoManagedClusterReconciler) mutateElasticsearchSecret(secret *corev1.Secret, secretData map[string][]byte) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = secretData
	return nil
}

// Get the Elasticsearch URL.
func (r *VerrazzanoManagedClusterReconciler) getElasticsearchURL() (URL string, err error) {
	var Ingress k8net.Ingress
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      vmiIngest,
	}
	if err := r.Get(context.TODO(), nsn, &Ingress); err != nil {
		return "", fmt.Errorf("Failed to fetch the VMI ingress %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	if len(Ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("VMI ingress %s/%s missing host entry in rule", nsn.Namespace, nsn.Name)
	}
	host := Ingress.Spec.Rules[0].Host
	if len(Ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("VMI ingress %s/%s host field is empty", nsn.Namespace, nsn.Name)
	}
	return fmt.Sprintf("https://%s:443", host), nil
}

// Get the Verrazzano secret
func (r *VerrazzanoManagedClusterReconciler) getVzSecret() (corev1.Secret, error) {
	var secret corev1.Secret
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.Verrazzano,
	}
	if err := r.Get(context.TODO(), nsn, &secret); err != nil {
		return corev1.Secret{}, fmt.Errorf("Failed to fetch the secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	return secret, nil
}

// Get the system-tls secret
func (r *VerrazzanoManagedClusterReconciler) getTLSSecret() (corev1.Secret, error) {
	var secret corev1.Secret
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.SystemTLS,
	}
	if err := r.Get(context.TODO(), nsn, &secret); err != nil {
		return corev1.Secret{}, fmt.Errorf("Failed to fetch the secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	return secret, nil
}

// GetElasticsearchSecretName returns the elasticsearch secret name
func GetElasticsearchSecretName(vmcName string) string {
	const suffix = "-elasticsearch"
	return generateManagedResourceName(vmcName) + suffix
}
