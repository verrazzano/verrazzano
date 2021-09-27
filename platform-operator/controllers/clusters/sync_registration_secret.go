// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	k8net "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const vmiIngest = "vmi-system-es-ingest"
const defaultElasticURL = "http://vmi-system-es-ingest-oidc:8775"
const defaultSecretName = "verrazzano"

// Create a registration secret with the managed cluster information.  This secret will
// be used on the managed cluster to get information about itself, like the cluster name
func (r *VerrazzanoManagedClusterReconciler) syncRegistrationSecret(vmc *clusterapi.VerrazzanoManagedCluster) error {
	secretName := GetRegistrationSecretName(vmc.Name)
	managedNamespace := vmc.Namespace

	_, err := r.createOrUpdateRegistrationSecret(vmc, secretName, managedNamespace)
	if err != nil {
		return err
	}

	return nil
}

// Create or update the registration secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateRegistrationSecret(vmc *clusterapi.VerrazzanoManagedCluster, name string, namespace string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = namespace
	secret.Name = name

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateRegistrationSecret(&secret, vmc.Name)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret, setting the kubeconfig data
func (r *VerrazzanoManagedClusterReconciler) mutateRegistrationSecret(secret *corev1.Secret, manageClusterName string) error {
	secret.Type = corev1.SecretTypeOpaque

	// Get the fluentd configuration for ES URL and secret
	fluentdESURL, fluentdESSecretName, err := r.getVzESURLSecret()
	if err != nil {
		return err
	}

	// Decide which ES URL to use.
	// If the fluentd ELASTICSEARCH_URL is the default "http://vmi-system-es-ingest-oidc:8775", use VMI ES ingress URL.
	// If the fluentd ELASTICSEARCH_URL is not the default, meaning it is a custom ES, use the external ES URL.
	esURL := fluentdESURL
	if esURL == defaultElasticURL {
		esURL, err = r.getVmiESURL()
		if err != nil {
			return err
		}
	}

	// Get the CA bundle needed to connect to the admin keycloak
	adminCaBundle, err := r.getAdminCaBundle()
	if err != nil {
		return err
	}

	// Decide which ES secret to use for username/password and password.
	// If the fluentd elasticsearchSecret is the default "verrazzano", use VerrazzanoESInternal secret for username/password, and adminCaBundle for ES CA bundle.
	// if the fluentd elasticsearchSecret is not the default, meaning it is a custom secret, use its username/password and CA bundle.
	var esCaBundle []byte
	var esUsername []byte
	var esPassword []byte
	if fluentdESSecretName != "verrazzano" {
		esSecret, err := r.getSecret(fluentdESSecretName)
		if err != nil {
			return err
		}
		esCaBundle = esSecret.Data[FluentdESCaBundleKey]
		esUsername = esSecret.Data[VerrazzanoUsernameKey]
		esPassword = esSecret.Data[VerrazzanoPasswordKey]
	} else {
		esSecret, err := r.getSecret(constants.VerrazzanoESInternal)
		if err != nil {
			return err
		}
		esCaBundle = adminCaBundle
		esUsername = esSecret.Data[VerrazzanoUsernameKey]
		esPassword = esSecret.Data[VerrazzanoPasswordKey]
	}

	// Get the keycloak URL
	keycloakURL, err := r.getKeycloakURL()
	if err != nil {
		return err
	}

	// Build the secret data
	secret.Data = map[string][]byte{
		ManagedClusterNameKey:   []byte(manageClusterName),
		ESURLKey:                []byte(esURL),
		ESCaBundleKey:           esCaBundle,
		RegistrationUsernameKey: esUsername,
		RegistrationPasswordKey: esPassword,
		KeycloakURLKey:          []byte(keycloakURL),
		AdminCaBundleKey:        adminCaBundle,
	}
	return nil
}

// GetRegistrationSecretName returns the registration secret name
func GetRegistrationSecretName(vmcName string) string {
	const registrationSecretSuffix = "-registration"
	return generateManagedResourceName(vmcName) + registrationSecretSuffix
}

// getVzESURLSecret returns the elasticsearchURL and elasticsearchSecret from Verrazzano CR
func (r *VerrazzanoManagedClusterReconciler) getVzESURLSecret() (string, string, error) {
	url := defaultElasticURL
	secret := defaultSecretName
	vzList := vzapi.VerrazzanoList{}
	err := r.List(context.TODO(), &vzList, &client.ListOptions{})
	if err != nil {
		r.log.Error(err, "Can not list Verrazzano CR")
		return url, secret, err
	}
	// what to do when there is more than one Verrazzano CR
	for _, vz := range vzList.Items {
		if vz.Spec.Components.Fluentd != nil {
			if len(vz.Spec.Components.Fluentd.ElasticsearchURL) > 0 {
				url = vz.Spec.Components.Fluentd.ElasticsearchURL
			}
			if len(vz.Spec.Components.Fluentd.ElasticsearchSecret) > 0 {
				secret = vz.Spec.Components.Fluentd.ElasticsearchSecret
			}
		}
	}
	return url, secret, nil
}

// Get the VMI Elasticsearch URL.
func (r *VerrazzanoManagedClusterReconciler) getVmiESURL() (URL string, err error) {
	var Ingress k8net.Ingress
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      vmiIngest,
	}
	if err := r.Get(context.TODO(), nsn, &Ingress); err != nil {
		return "", fmt.Errorf("Failed to fetch the VMI ingress %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	if len(Ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("VMI ingress %s/%s missing host entry in rule", nsn.Namespace, nsn.Name)
	}
	host := Ingress.Spec.Rules[0].Host
	if len(Ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("VMI ingress %s/%s host field is empty", nsn.Namespace, nsn.Name)
	}
	return fmt.Sprintf("https://%s:443", host), nil
}

// Get secret from verrazzano-system namespace
func (r *VerrazzanoManagedClusterReconciler) getSecret(secretName string) (corev1.Secret, error) {
	var secret corev1.Secret
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      secretName,
	}
	if err := r.Get(context.TODO(), nsn, &secret); err != nil {
		return corev1.Secret{}, fmt.Errorf("Failed to fetch the secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	return secret, nil
}

// Get the CA bundle used by system-tls
func (r *VerrazzanoManagedClusterReconciler) getAdminCaBundle() ([]byte, error) {
	secret, err := r.getSecret(constants.SystemTLS)
	if err != nil {
		return nil, err
	}
	return secret.Data[CaCrtKey], nil
}

// Get the keycloak URL
func (r *VerrazzanoManagedClusterReconciler) getKeycloakURL() (string, error) {
	var ingress = &extv1beta1.Ingress{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: "keycloak", Namespace: "keycloak"}, ingress)
	if err != nil {
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", "keycloak", "keycloak", err)
	}
	return fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0]), nil
}
