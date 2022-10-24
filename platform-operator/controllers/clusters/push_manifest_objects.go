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
			// Do nothing if the condition is met in the status
			r.log.Debugf("The status condition %s is %s for VMC %s/%s, skipping the manifest objects push",
				clusterapi.ConditionManifestPushed, corev1.ConditionTrue, vmc.GetNamespace(), vmc.GetName())
			return true, nil
		}
	}
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		r.log.Debugf("Waiting to push manifest objects, Rancher ClusterID not found in the VMC %s/%s status", vmc.GetNamespace(), vmc.GetName())
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
		existingAgentSec, err := r.getSecret(vmc.Namespace, GetAgentSecretName(vmc.Name), true)
		if err != nil {
			return err
		}
		agentSecret.Data = existingAgentSec.Data
		return nil
	}, r.log)
	if err != nil {
		return false, err
	}
	err = createOrUpdateSecretRancherProxy(&regSecret, rc, clusterID, func() error {
		existingRegSecret, err := r.getSecret(vmc.Namespace, GetRegistrationSecretName(vmc.Name), true)
		if err != nil {
			return err
		}
		regSecret.Data = existingRegSecret.Data
		return nil
	}, r.log)
	if err != nil {
		return false, err
	}
	return true, nil
}
