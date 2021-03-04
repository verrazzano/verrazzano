// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzk8s "github.com/verrazzano/verrazzano/platform-operator/internal/k8s"

	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	kubeconfigKey         = "admin-kubeconfig"
	managedClusterNameKey = "managed-cluster-name"
)

// Needed for unit testing
var getConfigFunc = ctrl.GetConfig

func setConfigFunc(f func() (*rest.Config, error)) {
	getConfigFunc = f
}

// Create an admin secret with a kubeconfig that has a token allowing access to the managed cluster
// with restricted access as defined in the verrazzano-managed-cluster role.
// The code does the following:
//   1. get the service account for the managed cluster
//   2. get the name of the service account token from the service account secret name field
//   3. get the in-memory client configuration used to access the admin cluster
//   4. build a kubeconfig struct using data from the client config and the service account token
//   5. save the kubeconfig as a secret
//   6. update VMC with the admin secret name
func (r *VerrazzanoManagedClusterReconciler) syncAdminSecret(vmc *clusterapi.VerrazzanoManagedCluster) error {
	// These names are used internally in the generated kubeconfig. The names
	// are meant to be descriptive and the actual values don't affect behavior.
	const (
		clusterName = "admin"
		userName    = "mcAgent"
		contextName = "defaultContext"
	)

	// The same managed name and  vmc namespace is used for the service account and the kubeconfig secret,
	// for clarity use different vars
	saName := generateManagedResourceName(vmc.Name)
	secretName := GetAdminSecretName(vmc.Name)
	managedNamespace := vmc.Namespace

	// Get the service account
	var sa corev1.ServiceAccount
	saNsn := types.NamespacedName{
		Namespace: managedNamespace,
		Name:      saName,
	}
	if err := r.Get(context.TODO(), saNsn, &sa); err != nil {
		return fmt.Errorf("Failed to fetch the service account for VMC %s/%s, %v", managedNamespace, saName, err)
	}
	if len(sa.Secrets) == 0 {
		return fmt.Errorf("Service account %s/%s is missing a secret name", managedNamespace, saName)
	}

	// Get the service account token from the secret
	tokenName := sa.Secrets[0].Name
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: managedNamespace,
		Name:      tokenName,
	}
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		return fmt.Errorf("Failed to fetch the service account secret %s/%s, %v", managedNamespace, tokenName, err)
	}

	// Get client config, this has some of the info needed to build a kubeconfig
	config, err := getConfigFunc()
	if err != nil {
		return err
	}

	// Load the kubeconfig struct
	token := secret.Data["token"]
	b64Cert, err := getB64CAData(config)
	if err != nil {
		return err
	}
	serverURL, err := vzk8s.GetAPIServerURL(r)
	if err != nil {
		return err
	}

	kb := vzk8s.KubeconfigBuilder{
		ClusterName: clusterName,
		Server:      serverURL,
		CertAuth:    b64Cert,
		UserName:    userName,
		UserToken:   string(token),
		ContextName: contextName,
	}
	kc := kb.Build()

	// Convert the kubeconfig to yaml then write it to a secret
	kcBytes, err := yaml.Marshal(kc)
	if err != nil {
		return err
	}
	_, err = r.createOrUpdateAdminSecret(vmc, string(kcBytes), secretName, managedNamespace)
	if err != nil {
		return err
	}

	return nil
}

// Create or update the kubeconfig secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateAdminSecret(vmc *clusterapi.VerrazzanoManagedCluster, kubeconfig string, name string, namespace string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = namespace
	secret.Name = name

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateAdminSecret(&secret, kubeconfig, vmc.Name)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret, setting the kubeconfig data
func (r *VerrazzanoManagedClusterReconciler) mutateAdminSecret(secret *corev1.Secret, kubeconfig string, manageClusterName string) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		kubeconfigKey: []byte(kubeconfig),
	}
	return nil
}

// Get the CAData from memory or a file
func getB64CAData(config *rest.Config) (string, error) {
	if len(config.CAData) > 0 {
		return base64.StdEncoding.EncodeToString(config.CAData), nil
	}
	s, err := ioutil.ReadFile(config.CAFile)
	if err != nil {
		return "", fmt.Errorf("Error %v reading CAData file %s", err, config.CAFile)
	}
	return base64.StdEncoding.EncodeToString(s), nil
}

// GetAdminSecretName returns the admin secret name
func GetAdminSecretName(vmcName string) string {
	const adminSecretSuffix = "-admin"
	return generateManagedResourceName(vmcName) + adminSecretSuffix
}
