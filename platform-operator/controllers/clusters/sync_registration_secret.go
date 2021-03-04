// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Create a registration secret with the managed cluster information.  This secret will
// be used on the managed cluster to get information about itself, like the cluster name
func (r *VerrazzanoManagedClusterReconciler) syncRegistrationSecret(vmc *clusterapi.VerrazzanoManagedCluster) error {
	// These names are used internally in the generated kubeconfig. The names
	// are meant to be descriptive and the actual values don't affect behavior.
	const (
		clusterName = "admin"
		userName    = "mcAgent"
		contextName = "defaultContext"
	)
	secretName := GetRegistrationSecretName(vmc.Name)
	managedNamespace := vmc.Namespace

	_, err := r.createOrUpdateRegistrationSecret(vmc, secretName, managedNamespace)
	if err != nil {
		return err
	}

	return nil
}

// Create or update the kubeconfig secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateRegistrationSecret(vmc *clusterapi.VerrazzanoManagedCluster, name string, namespace string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = namespace
	secret.Name = name

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateRegistrationSecret(&secret, vmc.Name)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret, setting the kubeconfig data
func (r *VerrazzanoManagedClusterReconciler) mutateRegistrationSecret(secret *corev1.Secret, manageClusterName string) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		managedClusterNameKey: []byte(manageClusterName),
	}
	return nil
}

// GetRegistrationSecretName returns the registration secret name
func GetRegistrationSecretName(vmcName string) string {
	const registrationSecretSuffix = "-registration"
	return generateManagedResourceName(vmcName) + registrationSecretSuffix
}
