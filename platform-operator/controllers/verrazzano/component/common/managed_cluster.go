// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetManagedClusterRegistrationSecret fetches the managed cluster registration secret if it exists.
// returns nil if the secret is not found.
func GetManagedClusterRegistrationSecret(client clipkg.Client) (*corev1.Secret, error) {
	registrationSecret := &corev1.Secret{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: constants.MCRegistrationSecret, Namespace: constants.VerrazzanoSystemNamespace}, registrationSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if !isManaged(registrationSecret) {
		return nil, nil
	}
	return registrationSecret, nil
}

func isManaged(secret *corev1.Secret) bool {
	return secret.ResourceVersion != "" && secret.Data != nil
}
