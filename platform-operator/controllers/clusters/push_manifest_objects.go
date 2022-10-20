// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
)

// pushManifestObjects applies the Verrazzano manifest objects to the managed cluster.
// To access the managed cluster, we are taking advantage of the Rancher proxy
func (r *VerrazzanoManagedClusterReconciler) pushManifestObjects(vmc *clusterapi.VerrazzanoManagedCluster) (bool, error) {
	for _, condition := range vmc.Status.Conditions {
		if condition.Type == clusterapi.ConditionManifestPushed && condition.Status == corev1.ConditionTrue {
			r.log.Once("Manifest has been successfully pushed, skipping the push process")
			return true, nil
		}
	}
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		return false, nil
	}
	rc, err := newRancherConfig(r.Client, r.log)
	if err != nil || rc == nil {
		return false, err
	}

	// If the managed cluster is not active, we should not attempt to push resources
	if isActive, err := isManagedClusterActiveInRancher(rc, clusterID, r.log); !isActive || err != nil {
		return false, err
	}

	// Create or Update the agent and registration secrets
	agentSecret := corev1.Secret{}
	agentSecret.Namespace = constants.VerrazzanoSystemNamespace
	agentSecret.Name = constants.MCAgentSecret
	regSecret := corev1.Secret{}
	regSecret.Namespace = constants.VerrazzanoSystemNamespace
	regSecret.Name = constants.MCRegistrationSecret
	err = createOrUpdateSecretRancherProxy(&agentSecret, rc, clusterID, func() error {
		agentSecret, err = r.getSecret(vmc.Namespace, GetAgentSecretName(vmc.Name), true)
		if err != nil {
			return err
		}
		// Reset the names to their original values if they get overwritten
		agentSecret.Namespace = constants.VerrazzanoSystemNamespace
		agentSecret.Name = constants.MCAgentSecret
		return nil
	}, r.log)
	if err != nil {
		return false, err
	}
	err = createOrUpdateSecretRancherProxy(&regSecret, rc, clusterID, func() error {
		regSecret, err = r.getSecret(vmc.Namespace, GetRegistrationSecretName(vmc.Name), true)
		if err != nil {
			return err
		}
		// Reset the names to their original values if they get overwritten
		regSecret.Namespace = constants.VerrazzanoSystemNamespace
		regSecret.Name = constants.MCRegistrationSecret
		return nil
	}, r.log)
	if err != nil {
		return false, err
	}
	return true, nil
}
