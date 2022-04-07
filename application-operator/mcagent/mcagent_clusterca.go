// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Synchronize Secret objects to the local cluster
func (s *Syncer) syncClusterCAs() error {
	s.syncAdminClusterCA()
	s.syncLocalClusterCA()
	return nil
}

// Synchronize the admin cluster CA cert -- update local copy if admin CA changes
func (s *Syncer) syncAdminClusterCA() error {

	// Get the cluster CA secret from the admin cluster
	adminCASecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.ClusterCASecret,
	}, &adminCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Get the local cluster registration secret
	registrationSecret := corev1.Secret{}
	err = s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.MCRegistrationSecret,
	}, &registrationSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Update the local cluster registration secret if the admin CA certs are different
	if !bytes.Equal(registrationSecret.Data["ca-bundle"], adminCASecret.Data["ca.crt"]) {
		newSecret := corev1.Secret{}
		newSecret.Name = registrationSecret.Name
		newSecret.Namespace = registrationSecret.Namespace
		newSecret.Labels = registrationSecret.Labels
		newSecret.Annotations = registrationSecret.Annotations
		newSecret.Data = registrationSecret.Data
		newSecret.Data["ca-bundle"] = adminCASecret.Data["ca.crt"]
		_, err = controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &newSecret, func() error { return nil })
		if err != nil {
			s.Log.Errorw(fmt.Sprintf("Failed syncing admin CA certificate: %v", err),
				"Secret", registrationSecret.Name)
		}
	}

	return nil
}

// Synchronize the local cluster CA cert -- update admin copy if local CA changes
func (s *Syncer) syncLocalClusterCA() error {

	// Get the local cluster CA secret
	localCASecret := corev1.Secret{}
	err := s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.ClusterCASecret,
	}, &localCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Get the managed cluster CA secret from the admin cluster
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := platformopclusters.VerrazzanoManagedCluster{}
	err = s.AdminClient.Get(s.Context, vmcName, &vmc)
	if err != nil {
		return err
	}
	adminVMCCASecret := corev1.Secret{}
	err = s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      vmc.Spec.CASecret,
	}, &adminVMCCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Update the VMC cluster CA secret if the local CA is different
	if !bytes.Equal(adminVMCCASecret.Data["cacrt"], localCASecret.Data["ca.crt"]) {
		newSecret := corev1.Secret{}
		newSecret.Name = adminVMCCASecret.Name
		newSecret.Namespace = adminVMCCASecret.Namespace
		newSecret.Labels = adminVMCCASecret.Labels
		newSecret.Annotations = adminVMCCASecret.Annotations
		newSecret.Data = adminVMCCASecret.Data
		newSecret.Data["cacrt"] = localCASecret.Data["ca.crt"]
		_, err = controllerutil.CreateOrUpdate(s.Context, s.AdminClient, &newSecret, func() error { return nil })
		if err != nil {
			s.Log.Errorw(fmt.Sprintf("Failed syncing local CA certificate: %v", err),
				"Secret", adminVMCCASecret.Name)
		}
	}

	return nil
}
