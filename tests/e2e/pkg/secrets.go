// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/base64"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListSecrets returns the list of secrets in a given namespace for the cluster
func ListSecrets(namespace string) (*corev1.SecretList, error) {
	// Get the kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to list secrets in namespace %s with error: %v", namespace, err))
		return nil, err
	}
	return secrets, nil
}

// GetSecret returns the secret in a given namespace for the cluster specified in the environment
func GetSecret(namespace string, name string) (*corev1.Secret, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}
	return GetSecretInCluster(namespace, name, kubeconfigPath)
}

// GetSecretInCluster returns the secret in a given namespace for the given cluster
func GetSecretInCluster(namespace string, name string, kubeconfigPath string) (*corev1.Secret, error) {
	// Get the kubernetes clientset for the given cluster
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("GetSecretInCluster error: %s", err))
	}
	return secret, err
}

// CreateCredentialsSecret creates opaque secret
func CreateCredentialsSecret(namespace string, name string, username string, pw string, labels map[string]string) (*corev1.Secret, error) {
	return CreateCredentialsSecretFromMap(namespace, name, map[string]string{
		"password": pw,
		"username": username,
	}, labels)
}

// CreateCredentialsSecretInCluster creates opaque secret
func CreateCredentialsSecretInCluster(namespace string, name string, username string, pw string, labels map[string]string, kubeconfigPath string) (*corev1.Secret, error) {
	return CreateCredentialsSecretFromMapInCluster(namespace, name, map[string]string{
		"password": pw,
		"username": username,
	}, labels, kubeconfigPath)
}

// CreateCredentialsSecretFromMap creates opaque secret from the given map of values
func CreateCredentialsSecretFromMap(namespace string, name string, values, labels map[string]string) (*corev1.Secret, error) {
	Log(Info, fmt.Sprintf("CreateCredentialsSecret %s in %s", name, namespace))
	// Get the kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: values,
	}
	scr, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("CreateSecretOfOpaque %v error: %v", name, err))
	}
	return scr, err
}

// CreateCredentialsSecretFromMapInCluster creates opaque secret from the given map of values
func CreateCredentialsSecretFromMapInCluster(namespace string, name string, values, labels map[string]string, kubeconfigPath string) (*corev1.Secret, error) {
	Log(Info, fmt.Sprintf("CreateCredentialsSecret %s in %s", name, namespace))
	// Get the kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: values,
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
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}

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
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}

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
		if !errors.IsAlreadyExists(err) {
			Log(Error, fmt.Sprintf("CreateDockerSecret %v error: %v", name, err))
			return nil, err
		}
		Log(Info, fmt.Sprintf("Secret %s/%s already exists, updating", namespace, name))
		return clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	}
	return scr, err
}

// CreateDockerSecretInCluster creates docker secret
func CreateDockerSecretInCluster(namespace string, name string, server string, username string, password string, kubeconfigPath string) (*corev1.Secret, error) {
	Log(Info, fmt.Sprintf("CreateDockerSecret %s in %s", name, namespace))
	// Get the kubernetes clientset
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset with error: %v", err))
		return nil, err
	}

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
		if !errors.IsAlreadyExists(err) {
			Log(Error, fmt.Sprintf("CreateDockerSecretInCluster %v error: %v", name, err))
			return nil, err
		}
		Log(Info, fmt.Sprintf("CreateDockerSecretInCluster secret %s/%s already exists, updating", namespace, name))
		return clientset.CoreV1().Secrets(namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	}
	return scr, err
}

// DeleteSecret deletes the specified secret in the specified namespace
func DeleteSecret(namespace string, name string) error {
	// Get the kubernetes clientset
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return nil
	}
	return clientset.CoreV1().Secrets(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// SecretsCreated checks if all the secrets identified by names are created
func SecretsCreated(namespace string, names ...string) bool {
	secrets, err := ListSecrets(namespace)
	if err != nil {
		return false
	}
	missing := missingSecrets(secrets.Items, names...)
	Log(Info, fmt.Sprintf("Secrets %v were NOT created in %v", missing, namespace))
	return len(missing) == 0
}

func missingSecrets(secrets []corev1.Secret, namePrefixes ...string) []string {
	var missing []string
	for _, name := range namePrefixes {
		if !secretExists(secrets, name) {
			missing = append(missing, name)
		}
	}
	return missing
}

func secretExists(secrets []corev1.Secret, namePrefix string) bool {
	for i := range secrets {
		if strings.HasPrefix(secrets[i].Name, namePrefix) {
			return true
		}
	}
	return false
}

// CreateSecret creates the given secret
func CreateSecret(secret *corev1.Secret) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	return err
}

// UpdateSecret updates the given secret
func UpdateSecret(secret *corev1.Secret) error {
	clientset, err := k8sutil.GetKubernetesClientset()
	if err != nil {
		return err
	}
	_, err = clientset.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	return err
}
