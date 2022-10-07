// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzk8s "github.com/verrazzano/verrazzano/platform-operator/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// Needed for unit testing
var getConfigFunc = ctrl.GetConfig

func setConfigFunc(f func() (*rest.Config, error)) {
	getConfigFunc = f
}

// rancherBasedKubeConfigEnabled - feature flag for whether we support
// Rancher based Kubeconfig - introduced for VZ-6448 Populate VMC with managed cluster data
// Should be removed and enabled by default when story VZ-6449
var rancherBasedKubeConfigEnabled = false

// Create an agent secret with a kubeconfig that has a token allowing access to the managed cluster
// with restricted access as defined in the verrazzano-managed-cluster role.
// The code does the following:
//  1. get the service account for the managed cluster
//  2. get the name of the service account token from the service account secret name field
//  3. get the in-memory client configuration used to access the admin cluster
//  4. build a kubeconfig struct using data from the client config and the service account token
//  5. save the kubeconfig as a secret
//  6. update VMC with the admin secret name
func (r *VerrazzanoManagedClusterReconciler) syncAgentSecret(vmc *clusterapi.VerrazzanoManagedCluster) error {
	// The same managed name and  vmc namespace is used for the service account and the kubeconfig secret,
	// for clarity use different vars
	saName := generateManagedResourceName(vmc.Name)
	secretName := GetAgentSecretName(vmc.Name)
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
	var tokenName string
	if len(sa.Secrets) == 0 {
		r.log.Oncef("Service account %s/%s is missing a secret name. Using the service account token secret created"+
			" by the VerrazzanoManagedCluster controller", managedNamespace, saName)
		tokenName = sa.Name + "-token"
	} else {
		// Get the service account token from the secret
		tokenName = sa.Secrets[0].Name
	}

	var serviceAccountSecret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: managedNamespace,
		Name:      tokenName,
	}
	if err := r.Get(context.TODO(), secretNsn, &serviceAccountSecret); err != nil {
		return fmt.Errorf("Failed to fetch the service account secret %s/%s, %v", managedNamespace, tokenName, err)
	}

	// Build the kubeconfig
	var err error
	var kc *vzk8s.KubeConfig
	// Check feature flag before building kubeconfig from Rancher - feature flag was introduced in
	// VZ-6448, to be removed when VZ-6449 is completed
	if rancherBasedKubeConfigEnabled {
		kc, err = r.buildKubeConfigUsingRancherURL(serviceAccountSecret)
	}
	if err != nil || !rancherBasedKubeConfigEnabled {
		r.log.Oncef("Failed to build admin kubeconfig using Rancher URL: %v", err)
		kc, err = r.buildKubeConfigUsingAdminConfigMap(serviceAccountSecret)
	}
	if err != nil {
		return fmt.Errorf("Failed to create kubeconfig for cluster %s: %v", vmc.Name, err)
	}

	// Convert the kubeconfig to yaml then write it to a secret
	kcBytes, err := yaml.Marshal(kc)
	if err != nil {
		return err
	}
	_, err = r.createOrUpdateAgentSecret(vmc, string(kcBytes), secretName, managedNamespace)
	if err != nil {
		return err
	}

	return nil
}

// buildKubeConfigUsingRancherURL builds the kubeconfig using the Rancher URL as the api server, and
// the CA cert of the Rancher ingress as the cert authority
func (r *VerrazzanoManagedClusterReconciler) buildKubeConfigUsingRancherURL(serviceAccountSecret corev1.Secret) (*vzk8s.KubeConfig, error) {
	vz, err := r.getVerrazzanoResource()
	if err != nil {
		return nil, r.log.ErrorfNewErr("Could not find Verrazzano resource")
	}
	if vz.Status.VerrazzanoInstance == nil {
		return nil, r.log.ErrorfNewErr("No instance information found in Verrazzano resource status")
	}
	rancherURL := vz.Status.VerrazzanoInstance.RancherURL
	if rancherURL == nil {
		return nil, fmt.Errorf("No Rancher URL found in Verrazzano resource status")
	}
	caCert, err := r.getRancherCACert()
	if err != nil {
		return nil, err
	}
	userToken := serviceAccountSecret.Data[mcconstants.TokenKey]
	return buildAdminKubeConfig(*rancherURL, caCert, string(userToken))
}

// buildKubeConfig builds the kubeconfig from the user-provided admin cluster URL and
// CA cert of the local cluster configuration
func (r *VerrazzanoManagedClusterReconciler) buildKubeConfigUsingAdminConfigMap(serviceAccountSecret corev1.Secret) (*vzk8s.KubeConfig, error) {
	// Get client config, this has some of the info needed to build a kubeconfig
	config, err := getConfigFunc()
	if err != nil {
		return nil, fmt.Errorf("Failed to get the client config, %v", err)
	}

	token := serviceAccountSecret.Data[mcconstants.TokenKey]
	b64Cert, err := getB64CAData(config)
	if err != nil {
		return nil, err
	}
	serverURL, err := vzk8s.GetAPIServerURL(r.Client)
	if err != nil {
		return nil, err
	}
	return buildAdminKubeConfig(serverURL, b64Cert, string(token))
}

// buildAdminKubeConfig builds the kubeconfig for the admin i.e. local cluster, given the API server
// URL, user credentials and server CA data
func buildAdminKubeConfig(serverURL string, caCert string, userToken string) (*vzk8s.KubeConfig, error) {
	// These names are used internally in the generated kubeconfig. The names
	// are meant to be descriptive and the actual values don't affect behavior.
	const (
		clusterName = "admin"
		userName    = "mcAgent"
		contextName = "defaultContext"
	)

	// Load the kubeconfig struct and build the kubeconfig
	kb := vzk8s.KubeconfigBuilder{
		ClusterName: clusterName,
		Server:      serverURL,
		CertAuth:    caCert,
		UserName:    userName,
		UserToken:   userToken,
		ContextName: contextName,
	}
	kc := kb.Build()
	return &kc, nil
}

// Create or update the kubeconfig secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateAgentSecret(vmc *clusterapi.VerrazzanoManagedCluster, kubeconfig string, name string, namespace string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = namespace
	secret.Name = name

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateAgentSecret(&secret, kubeconfig, vmc.Name)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret, setting the kubeconfig data
func (r *VerrazzanoManagedClusterReconciler) mutateAgentSecret(secret *corev1.Secret, kubeconfig string, manageClusterName string) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		mcconstants.KubeconfigKey:         []byte(kubeconfig),
		mcconstants.ManagedClusterNameKey: []byte(manageClusterName),
	}
	return nil
}

// getRancherCACert returns the certificate authority data from Rancher's TLS secret
func (r *VerrazzanoManagedClusterReconciler) getRancherCACert() (string, error) {
	ingressSecret := corev1.Secret{}

	err := r.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: vzconst.RancherSystemNamespace,
		Name:      vzconst.AdditionalTLS,
	}, &ingressSecret)
	if client.IgnoreNotFound(err) != nil {
		return "", err
	}

	var caData []byte
	if err == nil {
		caData = ingressSecret.Data[vzconst.AdditionalTLSCAKey]
	} else {
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: rancherTLSSecret, Namespace: vzconst.RancherSystemNamespace}, &ingressSecret)
		if err != nil {
			return "", err
		}
		caData = ingressSecret.Data[mcconstants.CaCrtKey]
	}
	return base64.StdEncoding.EncodeToString(caData), nil
}

// Get the CAData from memory or a file
func getB64CAData(config *rest.Config) (string, error) {
	if len(config.CAData) > 0 {
		return base64.StdEncoding.EncodeToString(config.CAData), nil
	}
	s, err := os.ReadFile(config.CAFile)
	if err != nil {
		return "", fmt.Errorf("Error %v reading CAData file %s", err, config.CAFile)
	}
	return base64.StdEncoding.EncodeToString(s), nil
}

// GetAgentSecretName returns the admin secret name
func GetAgentSecretName(vmcName string) string {
	const suffix = "-agent"
	return generateManagedResourceName(vmcName) + suffix
}
