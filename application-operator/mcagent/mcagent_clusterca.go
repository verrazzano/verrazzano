// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	keyCaCrtNoDot = "cacrt"
	keyCaCrt      = "ca.crt"
	keyCaBundle   = "ca-bundle"
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

	// Get the cluster CA secret from the admin cluster
	adminCASecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      constants.VerrazzanoLocalCABundleSecret,
	}, &adminCASecret)
	if err != nil {
		return err
	}

	// Get the local cluster registration secret
	registrationSecret := corev1.Secret{}
	err = s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.MCRegistrationSecret,
	}, &registrationSecret)
	if err != nil {
		return err
	}

	// Update the local cluster registration secret if the admin CA certs are different
	if !secretsEqualTrimmedWhitespace(registrationSecret.Data[keyCaBundle], adminCASecret.Data[keyCaBundle]) {
		result, err := controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &registrationSecret, func() error {
			registrationSecret.Data[keyCaBundle] = adminCASecret.Data[keyCaBundle]
			return nil
		})
		if err != nil {
			s.Log.Errorw(fmt.Sprintf("Failed syncing admin CA certificate: %v", err),
				"Secret", registrationSecret.Name)
		} else {
			s.Log.Infof("Updated local cluster registration secret, result was: %v", result)
		}
	}

	return nil
}

// syncLocalClusterCA - synchronize the local cluster CA cert -- update admin copy if local CA changes
func (s *Syncer) syncLocalClusterCA() error {

	// Get the local cluster CA secret
	localCASecret := corev1.Secret{}
	err := s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.VerrazzanoIngressTLSSecret,
	}, &localCASecret)
	if err != nil {
		return err
	}

	// Get the managed cluster CA secret from the admin cluster
	vmc := platformopclusters.VerrazzanoManagedCluster{}
	err = s.AdminClient.Get(s.Context, client.ObjectKey{
		Name:      s.ManagedClusterName,
		Namespace: constants.VerrazzanoMultiClusterNamespace,
	}, &vmc)
	if err != nil {
		return err
	}

	vmcCASecret := corev1.Secret{}
	err = s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      vmc.Spec.CASecret,
	}, &vmcCASecret)
	if err != nil {
		return err
	}

	// Update the VMC cluster CA secret if the local CA is different
	if !secretsEqualTrimmedWhitespace(vmcCASecret.Data[keyCaCrtNoDot], localCASecret.Data[keyCaCrt]) {
		result, err := controllerutil.CreateOrUpdate(s.Context, s.AdminClient, &vmcCASecret, func() error {
			vmcCASecret.Data[keyCaCrtNoDot] = localCASecret.Data[keyCaCrt]
			return nil
		})
		if err != nil {
			s.Log.Errorw(fmt.Sprintf("Failed syncing local CA certificate: %v", err),
				"Secret", vmcCASecret.Name)
		} else {
			s.Log.Infof("Updated VMC cluster CA secret on admin cluster, result was: %v", result)
		}
	}

	return nil
}

func secretsEqualTrimmedWhitespace(secret1, secret2 []byte) bool {
	a := bytes.Trim(secret1, " \t\n\r")
	b := bytes.Trim(secret2, " \t\n\r")
	return bytes.Equal(a, b)
}
