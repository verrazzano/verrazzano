// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const clusterSecretsNamespace = "verrazzano-system"
const clusterCASecret = "system-tls"
const clusterRegistrationSecret = "verrazzano-cluster-registration"

// Synchronize Secret objects to the local cluster
func (s *Syncer) syncClusterCAs() error {
	// Get the cluster CA secret from the admin cluster
	adminCASecret := corev1.Secret{}
	err := s.AdminClient.Get(s.Context, client.ObjectKey{
		Namespace: clusterSecretsNamespace,
		Name:      clusterCASecret,
	}, &adminCASecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Get the local cluster registration secret
	registrationSecret := corev1.Secret{}
	err = s.LocalClient.Get(s.Context, client.ObjectKey{
		Namespace: clusterSecretsNamespace,
		Name:      clusterRegistrationSecret,
	}, &registrationSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	// Update the local cluster registration secret if the admin CA certs are different
	if !bytes.Equal(registrationSecret.Data["ca-bundle"], adminCASecret.Data["ca.crt"]) {
		newSecret := corev1.Secret{}
		newSecret.Name = registrationSecret.Name
		newSecret.Namespace = registrationSecret.Namespace
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
