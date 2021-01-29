// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/onsi/ginkgo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListSecrets returns the list of secrets in a given namespace for the cluster
func ListSecrets(namespace string) *corev1.SecretList {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to list secrets in namespace %s with error: %v", namespace, err))
	}
	return secrets
}

// GetSecret returns the a secret in a given namespace for the cluster
func GetSecret(namespace string, name string) (*corev1.Secret, error) {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		ginkgo.Fail(fmt.Sprintf("Failed to get secrets %s in namespace %s with error: %v", name, namespace, err))
	}
	return secret, err
}

// CreateCredentialsSecret creates opaque secret
func CreateCredentialsSecret(namespace string, name string, username string, pw string, labels map[string]string) (*corev1.Secret, error) {
	Log(Info, fmt.Sprintf("CreateCredentialsSecret %s in %s", name, namespace))
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"password": pw,
			"username": username,
		},
	}
	scr, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("CreateSecretOfOpaque %v error: %v", name, err))
	}
	return scr, err
}

// CreatePasswordSecret creates opaque secret
func CreatePasswordSecret(namespace string, name string, pw string, labels map[string]string) (*corev1.Secret, error) {
	Log(Info, fmt.Sprintf("CreatePasswordSecret %s in %s", name, namespace))
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"password": pw,
		},
	}
	scr, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("CreatePasswordSecret %v error: %v", name, err))
	}
	return scr, err
}

// CreateDockerSecret creates docker secret
func CreateDockerSecret(namespace string, name string, server string, username string, password string) (*corev1.Secret, error) {
	Log(Info, fmt.Sprintf("CreateDockerSecret %s in %s", name, namespace))
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%v:%v", username, password)))
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		StringData: map[string]string{
			".dockerconfigjson": fmt.Sprintf(dockerconfigjsonTemplate, server, username, password, auth),
		},
	}
	scr, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("CreateDockerSecret %v error: %v", name, err))
	}
	return scr, err
}

func DeleteSecret(namespace string, name string) error {
	// Get the kubernetes clientset
	clientset := GetKubernetesClientset()
	return clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}
