// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"encoding/base64"
	"fmt"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// These kubeconfig related structs represent the kubeconfig information needed to build kubeconfig
// YAML.
// client connection to a cluster using a service account token
// The MC agents will use this kubeconfig to get access to the admin cluster
type kubeConfig struct {
	Clusters       []kcCluster `json:"clusters"`
	Users          []kcUser    `json:"users"`
	Contexts       []kcContext `json:"contexts"`
	CurrentContext string      `json:"current-context"`
}
type kcCluster struct {
	Name    string        `json:"name"`
	Cluster kcClusterData `json:"cluster"`
}
type kcClusterData struct {
	Server   string `json:"server"`
	CertAuth string `json:"certificate-authority-data"`
}
type kcUser struct {
	Name string     `json:"name"`
	User kcUserData `json:"user"`
}
type kcUserData struct {
	Token string `json:"token"`
}
type kcContext struct {
	Name    string        `json:"name"`
	Context kcContextData `json:"context"`
}
type kcContextData struct {
	User    string `json:"user"`
	Cluster string `json:"cluster"`
}

// These names are descriptive only and used internally in the genereated kubeconfig.
const clusterName = "admin"
const userName = "managed"
const contextName = "defaultContext"

var getConfigFunc = ctrl.GetConfig

// Create a kubecconfig that has a token that allows access to the managed cluster with restricted access as defined
// in the verrazzano-managed-cluster role.
// The code does the following:
//   1. get the service account for the managed cluster
//   2. get the name of the service account token from the service account secret naem field
//   3. get the in-memory client configuration used to access the admin cluster
//   4. build a kubeconfig struct using data from the client config and the service token
//   5. save the kubeconfig as a secret
func (r *VerrazzanoManagedClusterReconciler) reconcileKubeConfig(vmc *clusterapi.VerrazzanoManagedCluster) error {

	// The same managed name and  vmc namespace is used for the service account and the kubeconfig secret,
	// for clarity use different vars
	saName := generateManagedResourceName(vmc.Name)
	secretName := generateManagedResourceName(vmc.Name)
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
	config, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	// Load the kubeconfig struct and saved it to the secret.
	token := secret.Data["token"]
	b64Cert := base64.StdEncoding.EncodeToString(config.CAData)
	kc := kubeConfig{
		Clusters: []kcCluster{{
			Name: clusterName,
			Cluster: kcClusterData{
				Server:   config.Host,
				CertAuth: b64Cert,
			},
		}},
		Users: []kcUser{{
			Name: userName,
			User: kcUserData{
				Token: string(token),
			},
		}},
		Contexts: []kcContext{{
			Name: contextName,
			Context: kcContextData{
				User:    userName,
				Cluster: clusterName,
			},
		}},
		CurrentContext: contextName,
	}

	// Convert the kubeconfig to yaml, base64 encode it, then write to a secret
	kcBytes, err := yaml.Marshal(kc)
	if err != nil {
		return err
	}
	_, err = r.createOrUpdateSecret(vmc, string(kcBytes), secretName, managedNamespace)
	if err != nil {
		return err
	}

	// Save the KubeconfigSecret in the VMC
	vmc.Spec.KubeconfigSecret = secretName
	err = r.Update(context.TODO(), vmc)
	if err != nil {
		return err
	}

	return nil
}

func (r *VerrazzanoManagedClusterReconciler) createOrUpdateSecret(vmc *clusterapi.VerrazzanoManagedCluster, kubeconfig string, name string, namespace string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = namespace
	secret.Name = name

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateSecret(&secret, kubeconfig)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

func (r *VerrazzanoManagedClusterReconciler) mutateSecret(secret *corev1.Secret, b64KubeConfig string) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		"kubeconfig": []byte(b64KubeConfig),
	}
	return nil
}
