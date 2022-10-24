// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"fmt"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	keyCaCrtNoDot = "cacrt"
)

// Synchronize Secret objects to the local cluster
func (s *Syncer) syncClusterCAs() (controllerutil.OperationResult, error) {
	managedClusterResult, err := s.syncRegistrationFromAdminCluster()
	if err != nil {
		s.Log.Errorf("Error syncing Admin Cluster CA: %v", err)
	}
	err = s.syncLocalClusterCA()
	if err != nil {
		s.Log.Errorf("Error syncing Local Cluster CA: %v", err)
	}
	return managedClusterResult, nil
}

// syncRegistrationFromAdminCluster - synchronize the admin cluster registration info including
// CA cert, URLs and credentials -- update local registration if any of those change
func (s *Syncer) syncRegistrationFromAdminCluster() (controllerutil.OperationResult, error) {

	opResult := controllerutil.OperationResultNone
	// Get the cluster CA secret from the admin cluster - for the CA secret, this is considered
	// the source of truth
	adminCASecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      constants.VerrazzanoLocalCABundleSecret,
	}, &adminCASecret)
	if err != nil {
		return opResult, err
	}

	// Get the managed cluster registration secret for THIS managed cluster, from the admin cluster.
	// This will be used to sync registration information here on the managed cluster.
	adminRegistrationSecret := corev1.Secret{}
	err = s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      getRegistrationSecretName(s.ManagedClusterName),
	}, &adminRegistrationSecret)
	if err != nil {
		return opResult, err
	}

	// Get the local cluster registration secret
	registrationSecret := corev1.Secret{}
	err = s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.MCRegistrationSecret,
	}, &registrationSecret)
	if err != nil {
		return opResult, err
	}

	// Update the local cluster registration secret if the admin CA certs are different, or if
	// any of the registration info on admin cluster is different
	if !byteSlicesEqualTrimmedWhitespace(registrationSecret.Data[mcconstants.AdminCaBundleKey], adminCASecret.Data[mcconstants.AdminCaBundleKey]) ||
		!registrationInfoEqual(registrationSecret, adminRegistrationSecret) {
		opResult, err = controllerutil.CreateOrUpdate(s.Context, s.LocalClient, &registrationSecret, func() error {
			// Get CA info from admin CA secret
			registrationSecret.Data[mcconstants.AdminCaBundleKey] = adminCASecret.Data[mcconstants.AdminCaBundleKey]

			// Get other registration info from admin registration secret for this managed cluster
			registrationSecret.Data[mcconstants.OSURLKey] = adminRegistrationSecret.Data[mcconstants.OSURLKey]
			registrationSecret.Data[mcconstants.RegistrationUsernameKey] = adminRegistrationSecret.Data[mcconstants.RegistrationUsernameKey]
			registrationSecret.Data[mcconstants.RegistrationPasswordKey] = adminRegistrationSecret.Data[mcconstants.RegistrationPasswordKey]
			registrationSecret.Data[mcconstants.KeycloakURLKey] = adminRegistrationSecret.Data[mcconstants.KeycloakURLKey]
			registrationSecret.Data[mcconstants.OSCaBundleKey] = adminRegistrationSecret.Data[mcconstants.OSCaBundleKey]
			registrationSecret.Data[mcconstants.JaegerOSURLKey] = adminRegistrationSecret.Data[mcconstants.JaegerOSURLKey]
			registrationSecret.Data[mcconstants.JaegerOSUsernameKey] = adminRegistrationSecret.Data[mcconstants.JaegerOSUsernameKey]
			registrationSecret.Data[mcconstants.JaegerOSPasswordKey] = adminRegistrationSecret.Data[mcconstants.JaegerOSPasswordKey]
			registrationSecret.Data[mcconstants.JaegerOSTLSCAKey] = adminRegistrationSecret.Data[mcconstants.JaegerOSTLSCAKey]
			registrationSecret.Data[mcconstants.JaegerOSTLSCertKey] = adminRegistrationSecret.Data[mcconstants.JaegerOSTLSCertKey]
			registrationSecret.Data[mcconstants.JaegerOSTLSKey] = adminRegistrationSecret.Data[mcconstants.JaegerOSTLSKey]
			return nil
		})
		if err != nil {
			s.Log.Errorw(fmt.Sprintf("Failed syncing admin CA certificate: %v", err),
				"Secret", registrationSecret.Name)
		} else {
			s.Log.Infof("Updated local cluster registration secret, result was: %v", opResult)
		}
	}

	return opResult, nil
}

func registrationInfoEqual(regSecret1 corev1.Secret, regSecret2 corev1.Secret) bool {
	return byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.OSURLKey],
		regSecret2.Data[mcconstants.OSURLKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.KeycloakURLKey],
			regSecret2.Data[mcconstants.KeycloakURLKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.RegistrationUsernameKey],
			regSecret2.Data[mcconstants.RegistrationUsernameKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.RegistrationPasswordKey],
			regSecret2.Data[mcconstants.RegistrationPasswordKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.OSCaBundleKey],
			regSecret2.Data[mcconstants.OSCaBundleKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.JaegerOSURLKey],
			regSecret2.Data[mcconstants.JaegerOSURLKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.JaegerOSUsernameKey],
			regSecret2.Data[mcconstants.JaegerOSUsernameKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.JaegerOSPasswordKey],
			regSecret2.Data[mcconstants.JaegerOSPasswordKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.JaegerOSTLSCAKey],
			regSecret2.Data[mcconstants.JaegerOSTLSCAKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.JaegerOSTLSCertKey],
			regSecret2.Data[mcconstants.JaegerOSTLSCertKey]) &&
		byteSlicesEqualTrimmedWhitespace(regSecret1.Data[mcconstants.JaegerOSTLSKey],
			regSecret2.Data[mcconstants.JaegerOSTLSKey])
}

// syncLocalClusterCA - synchronize the local cluster CA cert -- update admin copy if local CA changes
func (s *Syncer) syncLocalClusterCA() error {
	localCASecretData, err := s.getLocalClusterCASecretData()
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
	if !byteSlicesEqualTrimmedWhitespace(vmcCASecret.Data[keyCaCrtNoDot], localCASecretData) {
		result, err := controllerutil.CreateOrUpdate(s.Context, s.AdminClient, &vmcCASecret, func() error {
			vmcCASecret.Data[keyCaCrtNoDot] = localCASecretData
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

// getLocalClusterCASecret gets the local cluster CA secret and returns the CA data in the secret
// This could be in the Additional TLS secret in Rancher NS (for Let's Encrypt staging CA) or
// the Verrazzano ingress TLS secret in Verrazzano System NS for other cases
func (s *Syncer) getLocalClusterCASecretData() ([]byte, error) {
	localCASecret := corev1.Secret{}
	errAddlTLS := s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: globalconst.RancherSystemNamespace,
		Name:      globalconst.AdditionalTLS,
	}, &localCASecret)
	if client.IgnoreNotFound(errAddlTLS) != nil {
		return nil, errAddlTLS
	}

	if errAddlTLS == nil {
		return localCASecret.Data[globalconst.AdditionalTLSCAKey], nil
	}
	// additional TLS secret not found, check for Verrazzano TLS secret
	err := s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.VerrazzanoIngressTLSSecret,
	}, &localCASecret)
	if err != nil {
		return nil, err
	}
	return localCASecret.Data[mcconstants.CaCrtKey], nil
}

func byteSlicesEqualTrimmedWhitespace(byteSlice1, byteSlice2 []byte) bool {
	a := bytes.Trim(byteSlice1, " \t\n\r")
	b := bytes.Trim(byteSlice2, " \t\n\r")
	return bytes.Equal(a, b)
}

// Generate the common name used by all resources specific to a given managed cluster
func generateManagedResourceName(clusterName string) string {
	return fmt.Sprintf("verrazzano-cluster-%s", clusterName)
}

// getRegistrationSecretName returns the registration secret name for a managed cluster on the admin
// cluster
func getRegistrationSecretName(clusterName string) string {
	const registrationSecretSuffix = "-registration"
	return generateManagedResourceName(clusterName) + registrationSecretSuffix
}
