// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	vzstring "github.com/verrazzano/verrazzano/pkg/string"

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
	// When placements are changed a forbidden error can be returned.  In this case,
	// we want to fall through and delete orphaned resources.
	if err != nil && !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
		return err
	}

	// Write each of the secrets that are targeted for the local cluster
	for _, mcAppConfig := range allAdminMCAppConfigs.Items {
		if s.isThisCluster(mcAppConfig.Spec.Placement) {
			for _, adminSecret := range mcAppConfig.Spec.Secrets {
				secret := corev1.Secret{}
				namespacedName := types.NamespacedName{Name: adminSecret, Namespace: namespace}
				err := s.AdminClient.Get(s.Context, namespacedName, &secret)
				if err != nil {
					return err
				}
				_, err = s.createOrUpdateSecret(secret, mcAppConfig.Name)
				if err != nil {
					s.Log.Errorw(fmt.Sprintf("Failed syncing object: %v", err),
						"Secret",
						types.NamespacedName{Namespace: secret.Namespace, Name: secret.Name})
				}
			}
		}
	}

	// Cleanup orphaned or no longer placed Secret resources.
	// Get the list of Secret resources on the local cluster and compare to the list received from the admin cluster.
	// The admin cluster is the source of truth.
	allLocalSecrets := corev1.SecretList{}
	listOptions = &client.ListOptions{Namespace: namespace}
	err = s.LocalClient.List(s.Context, &allLocalSecrets, listOptions)
	if err != nil {
		s.Log.Errorf("Failed to list Secrets on local cluster: %v", err)
		return nil
	}
	for i, secret := range allLocalSecrets.Items {
		appConfigs, found := secret.Labels[mcAppConfigsLabel]
		// Only look at the secrets we have synced
		if !found {
			continue
		}
		// Delete Secret object if it is no longer placed on this local cluster
		if !s.k8sSecretPlacedOnCluster(secret, &allAdminMCAppConfigs) {
			err := s.LocalClient.Delete(s.Context, &allLocalSecrets.Items[i])
			if err != nil {
				s.Log.Errorf("Failed to delete Secret with name %s and namespace %s: %v", secret.Name, secret.Namespace, err)
			}
		} else {
			// Update the secrets label if the secret was shared across app configs and one of the app configs
			// was deleted.
			secretAppConfigs := strings.Split(appConfigs, ",")
			var actualAppConfigs []string
			for _, mcAppConfig := range allAdminMCAppConfigs.Items {
				for _, cluster := range mcAppConfig.Spec.Placement.Clusters {
					if cluster.Name == s.ManagedClusterName {
						for _, appConfigSecret := range mcAppConfig.Spec.Secrets {
							// Save the name of the MultiClusterApplicationConfiguration if we have a secret match
							if appConfigSecret == secret.Name {
								actualAppConfigs = append(actualAppConfigs, mcAppConfig.Name)
							}
						}
					}
				}
			}
			if !reflect.DeepEqual(secretAppConfigs, actualAppConfigs) {
				secret.Labels[mcAppConfigsLabel] = strings.Join(actualAppConfigs, ",")
				err := s.LocalClient.Update(s.Context, &allLocalSecrets.Items[i])
				if err != nil {
					s.Log.Errorf("Failed to update Secret with name %s and namespace %s: %v", secret.Name, secret.Namespace, err)
				}
			}
		}
	}

	return nil
}

// Create or update a Secret
func (s *Syncer) createOrUpdateSecret(secret corev1.Secret, mcAppConfigName string) (controllerutil.OperationResult, error) {
	var secretNew corev1.Secret
	secretNew.Namespace = secret.Namespace
	secretNew.Name = secret.Name
	// Create or update on the local cluster
	return controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &secretNew, func() error {
		mutateSecret(s.ManagedClusterName, mcAppConfigName, secret, &secretNew)
		return nil
	})
}

// mutateSecret mutates the Secret to reflect the contents of the parent Secret
func mutateSecret(managedClusterName string, mcAppConfigName string, secret corev1.Secret, secretNew *corev1.Secret) {
	if secretNew.Labels == nil {
		secretNew.Labels = make(map[string]string)
		if secret.Labels != nil {
			secretNew.Labels = secret.Labels
		}
	}

	secretNew.Labels[mcAppConfigsLabel] = vzstring.AppendToCommaSeparatedString(secretNew.Labels[mcAppConfigsLabel], mcAppConfigName)
	secretNew.Labels[managedClusterLabel] = managedClusterName
	// Mark the secret synced from Admin cluster with verrazzano-managed=true, to distinguish
	// those directly created by user on managed cluster
	secretNew.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault
	secretNew.Annotations = secret.Annotations
	secretNew.Type = secret.Type
	secretNew.Immutable = secret.Immutable
	secretNew.Data = secret.Data
	secretNew.StringData = secret.StringData
}

// k8sSecretPlacedOnCluster returns boolean indicating if the secret is placed on the local cluster
func (s *Syncer) k8sSecretPlacedOnCluster(secret corev1.Secret, allAdminMCAppConfigs *clustersv1alpha1.MultiClusterApplicationConfigurationList) bool {
	for _, mcAppConfig := range allAdminMCAppConfigs.Items {
		// Both a matching application configuration label and a matching cluster label be found for the
		// secret to be placed on the local cluster.
		if vzstring.CommaSeparatedStringContains(secret.Labels[mcAppConfigsLabel], mcAppConfig.Name) {
			for _, cluster := range mcAppConfig.Spec.Placement.Clusters {
				if cluster.Name == secret.Labels[managedClusterLabel] {
					return true
				}
			}
		}
	}

	return false
}
