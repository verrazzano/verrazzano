// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installjob

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewServiceAccount returns a service account resource for installing Verrazzano
// namespace - namespace of verrazzano resource
// name - name of verrazzano resource
func NewServiceAccount(namespace string, name string, imagePullSecret string, labels map[string]string) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
	if len(imagePullSecret) > 0 {
		sa.ImagePullSecrets = []corev1.LocalObjectReference{{Name: imagePullSecret}}
	}
	return sa
}
