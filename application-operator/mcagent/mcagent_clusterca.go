// Copyright (c) 2022, Oracle and/or its affiliates.
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
	err := s.syncAdminClusterCA()
	if err != nil {
		s.Log.Errorf("Error syncing Admin Cluster CA: %v", err)
	}
	err = s.syncLocalClusterCA()
	if err != nil {
		s.Log.Errorf("Error syncing Local Cluster CA: %v", err)
	}
	return nil
}

// syncAdminClusterCA - synchronize the admin cluster CA cert -- update local copy if admin CA changes
func (s *Syncer) syncAdminClusterCA() error {

	s.Log.Info("Syncing AdminClusterCA ...")

	// Get the cluster CA secret from the admin cluster
	adminCASecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.ClusterCASecret,
	}, &adminCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	s.Log.Info("Got admin cluster CA secret")

	// Get the local cluster registration secret
	registrationSecret := corev1.Secret{}
	err = s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.MCRegistrationSecret,
	}, &registrationSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	s.Log.Info("Got local cluster registration secret")

	// Update the local cluster registration secret if the admin CA certs are different
	if !secretsEqualTrimmedWhitespace(registrationSecret.Data["ca-bundle"], adminCASecret.Data["ca.crt"]) {
		s.Log.Info("CAs are different -- updating")
		s.Log.Info("Local registration Secret: %v", registrationSecret.Data["ca-bundle"])
		s.Log.Info("Admin CA Secret: %v", adminCASecret.Data["ca.crt"])
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
		s.Log.Info("Updated local cluster registration secret")
	} else {
		s.Log.Info("CAs are the same -- not updating")
	}

	return nil
}

// syncLocalClusterCA - synchronize the local cluster CA cert -- update admin copy if local CA changes
func (s *Syncer) syncLocalClusterCA() error {

	s.Log.Info("Syncing LocalClusterCA ...")

	// Get the local cluster CA secret
	localCASecret := corev1.Secret{}
	err := s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.ClusterCASecret,
	}, &localCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	s.Log.Info("Got local cluster CA secret")

	// Get the managed cluster CA secret from the admin cluster
	vmcName := client.ObjectKey{Name: s.ManagedClusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}
	vmc := platformopclusters.VerrazzanoManagedCluster{}
	err = s.AdminClient.Get(s.Context, vmcName, &vmc)
	if err != nil {
		return err
	}
	s.Log.Info("Got VMC from admin cluster")
	adminVMCCASecret := corev1.Secret{}
	err = s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      vmc.Spec.CASecret,
	}, &adminVMCCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	s.Log.Info("Got VMC CA secret from admin cluster")

	// Update the VMC cluster CA secret if the local CA is different
	if !secretsEqualTrimmedWhitespace(adminVMCCASecret.Data["cacrt"], localCASecret.Data["ca.crt"]) {
		s.Log.Info("CAs are different -- updating")
		s.Log.Info("VMC CA Secret: %v", adminVMCCASecret.Data["cacrt"])
		s.Log.Info("Local CA Secret: %v", localCASecret.Data["ca.crt"])
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
		s.Log.Info("Updated VMC cluster CA secret on admin cluster")
	} else {
		s.Log.Info("CAs are the same -- not updating")
	}

	return nil
}

func secretsEqualTrimmedWhitespace(secret1, secret2 []byte) bool {
	a := bytes.Trim(secret1, " \t\n")
	b := bytes.Trim(secret2, " \t\n")
	return bytes.Equal(a, b)
}
