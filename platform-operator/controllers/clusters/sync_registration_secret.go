// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/constants"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const vmiIngest = "vmi-system-es-ingest"
const defaultElasticURL = "http://verrazzano-authproxy-elasticsearch:8775"
const defaultSecretName = "verrazzano"
const rancherCAAdditionalPem = "ca-additional.pem"

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
	// If the fluentd ELASTICSEARCH_URL is the default "http://verrazzano-authproxy-elasticsearch:8775", use VMI ES ingress URL.
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
		esSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, fluentdESSecretName, true)
		if err != nil {
			return err
		}
		esCaBundle = esSecret.Data[FluentdESCaBundleKey]
		esUsername = esSecret.Data[VerrazzanoUsernameKey]
		esPassword = esSecret.Data[VerrazzanoPasswordKey]
	} else {
		esSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.VerrazzanoESInternal, true)
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
		return "", fmt.Errorf("failed to fetch the VMI ingress %s/%s, %v", nsn.Namespace, nsn.Name, err)
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
func (r *VerrazzanoManagedClusterReconciler) getSecret(namespace string, secretName string, required bool) (corev1.Secret, error) {
	var secret corev1.Secret
	nsn := types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}
	err := r.Get(context.TODO(), nsn, &secret)
	if err != nil {
		if !required && errors.IsNotFound(err) {
			return corev1.Secret{}, err
		} else {
			return corev1.Secret{}, fmt.Errorf("failed to fetch the secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
		}
	}
	return secret, nil
}

// Get the CA bundle used by system-tls and the optional rancher-ca-additional secret
func (r *VerrazzanoManagedClusterReconciler) getAdminCaBundle() ([]byte, error) {
	caBundle := []byte{}
	secret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.SystemTLS, true)
	if err != nil {
		return nil, err
	}
	caBundle = secret.Data[CaCrtKey]

	// Append CA from additional-ca secret if it exists
	optSecret, err := r.getSecret(constants.RancherSystemNamespace, constants.AdditionalTLS, false)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		// Decode the SystemTLS bundle
		systemTLSBundle := make([]byte, base64.StdEncoding.DecodedLen(len(secret.Data[CaCrtKey])))
		_, err2 := base64.URLEncoding.Decode(systemTLSBundle, secret.Data[CaCrtKey])
		if err2 != nil {
			return nil, err2
		}

		// Decode the additional CA bundle
		additionalCABundle := make([]byte, base64.StdEncoding.DecodedLen(len(optSecret.Data[rancherCAAdditionalPem])))
		_, err3 := base64.URLEncoding.Decode(additionalCABundle, optSecret.Data[rancherCAAdditionalPem])
		if err3 != nil {
			return nil, err3
		}

		// Combine the two CA bundles
		combinedBundle := []byte{}
		combinedBundle = systemTLSBundle
		combinedBundle = append(combinedBundle, additionalCABundle...)

		// Encode the combined bundle
		newBundle := make([]byte, base64.StdEncoding.DecodedLen(len(combinedBundle)))
		base64.URLEncoding.Encode(newBundle, combinedBundle)
		caBundle = newBundle
	}

	return caBundle, nil
}

// Get the keycloak URL
func (r *VerrazzanoManagedClusterReconciler) getKeycloakURL() (string, error) {
	var ingress = &networkingv1.Ingress{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: "keycloak", Namespace: "keycloak"}, ingress)
	if err != nil {
		return "", fmt.Errorf("unable to fetch ingress %s/%s, %v", "keycloak", "keycloak", err)
	}
	return fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0]), nil
}
