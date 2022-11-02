// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// syncMCAgentDeleteResources deletes the managed cluster resources if the correlating admin VMC gets deleted
func (s *Syncer) syncDeregistration() error {
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := platformopclusters.VerrazzanoManagedCluster{}
	err := s.AdminClient.Get(s.Context, vmcName, &vmc)
	if client.IgnoreNotFound(err) != nil && !apierrors.IsUnauthorized(err) {
		s.Log.Errorf("Failed to get the VMC resources %s/%s from the admin cluster: %v", constants.VerrazzanoMultiClusterNamespace, s.ManagedClusterName, err)
		return err
	}
	if err == nil && vmc.DeletionTimestamp.IsZero() {
		s.Log.Debugf("VMC resource %s/%s has been found and is not being deleted, skipping the MC Agent deregistration", constants.VerrazzanoMultiClusterNamespace, s.ManagedClusterName)
		return nil
	}

	s.Log.Infof("Verrazzano Managed Cluster %s/%s has been deleted, cleaning up managed cluster resources", constants.VerrazzanoMultiClusterNamespace, s.ManagedClusterName)
	mcAgentSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCAgentSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	err = s.LocalClient.Delete(context.TODO(), &mcAgentSec)
	if client.IgnoreNotFound(err) != nil {
		s.Log.Errorf("Failed to delete the managed cluster agent secret %s/%s: %v", constants.MCAgentSecret, constants.VerrazzanoSystemNamespace, err)
		return err
	}

	mcRegSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	err = s.LocalClient.Delete(context.TODO(), &mcRegSec)
	if client.IgnoreNotFound(err) != nil {
		s.Log.Errorf("Failed to delete the managed cluster registration secret %s/%s: %v", constants.MCRegistrationSecret, constants.VerrazzanoSystemNamespace, err)
		return err
	}
	return nil
}
