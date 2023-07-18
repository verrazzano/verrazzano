// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloakutil

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	keycloakPodName    = "keycloak-0"
	keycloakSecretName = "keycloak-http"
)

// CreateTestKeycloakPod constructs and returns Keycloak pod
func CreateTestKeycloakPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakPodName,
			Namespace: constants.KeycloakNamespace,
		},
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
}

// CreateTestKeycloakLoginSecret onstructs and returns a secret with name keycloak-http
func CreateTestKeycloakLoginSecret() *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      keycloakSecretName,
			Namespace: constants.KeycloakNamespace,
		},
		Data: map[string][]byte{"password": []byte("password")},
	}
}
