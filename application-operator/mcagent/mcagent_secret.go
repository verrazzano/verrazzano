// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize Secret objects to the local cluster
func (s *Syncer) syncSecretObjects(namespace string) error {
	// Get all the MultiClusterApplicationConfiguration objects from the admin cluster
	allAdminMCAppConfigs := clustersv1alpha1.MultiClusterApplicationConfigurationList{}
	listOptions := &client.ListOptions{Namespace: namespace}
	err := s.AdminClient.List(s.Context, &allAdminMCAppConfigs, listOptions)
	if err != nil {
		return client.IgnoreNotFound(err)
	}

	// Write each of the secrets that are targeted for the local cluster
	for _, mcAppConfig := range allAdminMCAppConfigs.Items {
		if s.isThisCluster(mcAppConfig.Spec.Placement) {
			for _, adminSecret := range mcAppConfig.Spec.Secrets{
				secret := corev1.Secret{}
				namespacedName := types.NamespacedName{Name: adminSecret, Namespace: namespace}
				err := s.AdminClient.Get(s.Context, namespacedName, &secret)
				if err != nil {
					return err
				}
				_, err = s.createOrUpdateSecret(secret)
				if err != nil {
					s.Log.Error(err, "Error syncing object",
						"Secret",
						types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name})
				}
			}
		}
	}

	// Delete orphaned or no longer placed Secret resources.
	// Get the list of Secret resources on the local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
/*	allLocalSecrets := corev1.Secrets{}
	err = s.LocalClient.List(s.Context, &allLocalSecrets, listOptions)
	if err != nil {
		s.Log.Error(err, "failed to list Secrets on local cluster")
		return nil
	}
	for _, secret := range allLocalSecrets.Items {
		// Delete each Secret object that is no longer placed on this cluster
		if !s.secretPlacedOnCluster(&secret, &allAdminMCAppConfigs) {
			err := s.LocalClient.Delete(s.Context, &secret)
			if err != nil {
				s.Log.Error(err, fmt.Sprintf("failed to delete Secret with name %q and namespace %q", secret.Name, secret.Namespace))
			}
		}
	}
*/
	return nil
}

// Create or update a Secret
func (s *Syncer) createOrUpdateSecret(secret corev1.Secret) (controllerutil.OperationResult, error) {
	var secretNew corev1.Secret
	secretNew.Namespace = secret.Namespace
	secretNew.Name = secret.Name

	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &secretNew, func() error {
		mutateSecret(s.ManagedClusterName, secret, &secretNew)
		return nil
	})
}

// mutateSecret mutates the Secret to reflect the contents of the parent Secret
func mutateSecret(managedClusteName string, secret corev1.Secret, secretNew *corev1.Secret) {
	secretNew.Labels = secret.Labels
	if secretNew.Labels == nil {
		secretNew.Labels = make(map[string]string)
	}
	secretNew.Labels["verrazzano.io/managed-cluster"] = managedClusteName

	secretNew.Annotations = secret.Annotations
	secretNew.Type = secret.Type
	secretNew.Immutable = secret.Immutable
	secretNew.Data = secret.Data
	secretNew.StringData = secret.StringData
}

// secretPlacedOnCluster returns boolean indicating if the list contains the object with the specified name and namespace
// and indicates the object is placed on the local cluster
func (s *Syncer) secretPlacedOnCluster(mcAdminList *clustersv1alpha1.MultiClusterSecretList, name string, namespace string) bool {
	for _, item := range mcAdminList.Items {
		if item.Name == name && item.Namespace == namespace {
			return s.isThisCluster(item.Spec.Placement)
		}
	}
	return false
}
