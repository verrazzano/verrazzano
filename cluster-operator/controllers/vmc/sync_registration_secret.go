// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	clusterapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

const vmiIngest = "vmi-system-os-ingest"
const operatorOSIngress = "opensearch"
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
		err := r.mutateRegistrationSecret(&secret, vmc.Name)
		if err != nil {
			return err
		}
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret, setting the kubeconfig data
func (r *VerrazzanoManagedClusterReconciler) mutateRegistrationSecret(secret *corev1.Secret, manageClusterName string) error {
	secret.Type = corev1.SecretTypeOpaque

	vzList := vzapi.VerrazzanoList{}
	err := r.List(context.TODO(), &vzList, &client.ListOptions{})
	if err != nil {
		r.log.Errorf("Failed to list Verrazzano CR: %v", err)
		return err
	}
	if len(vzList.Items) < 1 {
		return fmt.Errorf("can not find Verrazzano CR")
	}

	// Get the fluentd configuration for ES URL and secret
	fluentdESURL, fluentdESSecretName, err := r.getVzESURLSecret(&vzList)
	if err != nil {
		return err
	}

	// Decide which ES URL to use.
	// If the fluentd OPENSEARCH_URL is the default "http://verrazzano-authproxy-opensearch:8775", use VMI ES ingress URL.
	// If the fluentd OPENSEARCH_URL is the default "http://verrazzano-authproxy-opensearch-logging:8775", use Operator OS ingress URL.
	// If the fluentd OPENSEARCH_URL is not the default, meaning it is a custom ES, use the external ES URL.
	esURL := fluentdESURL
	if esURL == constants.DefaultOpensearchURL {
		esURL, err = r.getESURL(vzList, vmiIngest)
		if err != nil {
			return err
		}
	} else if esURL == constants.DefaultOperatorOSURL || esURL == constants.DefaultOperatorOSURLWithNS {
		esURL, err = r.getESURL(vzList, operatorOSIngress)
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
	// If the fluentd opensearchSecret is the default "verrazzano", use VerrazzanoESInternal secret for username/password, and adminCaBundle for ES CA bundle.
	// if the fluentd opensearchSecret is not the default, meaning it is a custom secret, use its username/password and CA bundle.
	var esCaBundle []byte
	var esUsername []byte
	var esPassword []byte
	if fluentdESSecretName != "verrazzano" {
		esSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, fluentdESSecretName, true)
		if err != nil {
			return err
		}
		esCaBundle = esSecret.Data[mcconstants.FluentdESCaBundleKey]
		esUsername = esSecret.Data[mcconstants.VerrazzanoUsernameKey]
		esPassword = esSecret.Data[mcconstants.VerrazzanoPasswordKey]
	} else {
		esSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.VerrazzanoESInternal, true)
		if err != nil {
			return err
		}
		esCaBundle = adminCaBundle
		esUsername = esSecret.Data[mcconstants.VerrazzanoUsernameKey]
		esPassword = esSecret.Data[mcconstants.VerrazzanoPasswordKey]
	}

	// Get the keycloak URL
	keycloakURL, err := k8sutil.GetURLForIngress(r.Client, "keycloak", "keycloak", "https")
	if err != nil {
		return err
	}

	// Get the Jaeger OpenSearch related data if it exists
	jaegerStorage, err := r.getJaegerOpenSearchConfig(&vzList)
	if err != nil {
		return err
	}

	// Build the secret data
	secret.Data = map[string][]byte{
		mcconstants.ManagedClusterNameKey:   []byte(manageClusterName),
		mcconstants.ESURLKey:                []byte(esURL),
		mcconstants.ESCaBundleKey:           esCaBundle,
		mcconstants.RegistrationUsernameKey: esUsername,
		mcconstants.RegistrationPasswordKey: esPassword,
		mcconstants.KeycloakURLKey:          []byte(keycloakURL),
		mcconstants.AdminCaBundleKey:        adminCaBundle,
		mcconstants.JaegerOSURLKey:          []byte(jaegerStorage.URL),
		mcconstants.JaegerOSTLSCAKey:        jaegerStorage.CA,
		mcconstants.JaegerOSTLSKey:          jaegerStorage.TLSKey,
		mcconstants.JaegerOSTLSCertKey:      jaegerStorage.TLSCert,
		mcconstants.JaegerOSUsernameKey:     jaegerStorage.username,
		mcconstants.JaegerOSPasswordKey:     jaegerStorage.password,
	}
	return nil
}

// GetRegistrationSecretName returns the registration secret name
func GetRegistrationSecretName(vmcName string) string {
	const registrationSecretSuffix = "-registration"
	return generateManagedResourceName(vmcName) + registrationSecretSuffix
}

// getVzESURLSecret returns the opensearchURL and opensearchSecret from Verrazzano CR
func (r *VerrazzanoManagedClusterReconciler) getVzESURLSecret(vzList *vzapi.VerrazzanoList) (string, string, error) {
	url := constants.DefaultOpensearchURL
	secret := defaultSecretName
	// what to do when there is more than one Verrazzano CR
	for _, vz := range vzList.Items {
		if vz.Spec.Components.Fluentd != nil {
			if len(vz.Spec.Components.Fluentd.OpenSearchURL) > 0 {
				url = vz.Spec.Components.Fluentd.OpenSearchURL
			}
			if len(vz.Spec.Components.Fluentd.OpenSearchSecret) > 0 {
				secret = vz.Spec.Components.Fluentd.OpenSearchSecret
			}
		}
	}
	return url, secret, nil
}

// Get the opensearch URL.
func (r *VerrazzanoManagedClusterReconciler) getESURL(vzList vzapi.VerrazzanoList, ingressName string) (URL string, err error) {
	if len(vzList.Items) == 0 {
		return "", nil
	}
	if ingressName == vmiIngest && !vzcr.IsOpenSearchEnabled(&vzList.Items[0]) {
		return "", nil
	}
	var Ingress k8net.Ingress
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      ingressName,
	}
	if err := r.Get(context.TODO(), nsn, &Ingress); err != nil {
		return "", fmt.Errorf("failed to fetch the OpenSearch ingress %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	if len(Ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("OpenSearch ingress %s/%s missing host entry in rule", nsn.Namespace, nsn.Name)
	}
	host := Ingress.Spec.Rules[0].Host
	if len(Ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("OpenSearch ingress %s/%s host field is empty", nsn.Namespace, nsn.Name)
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
		}
		return corev1.Secret{}, fmt.Errorf("failed to fetch the secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	return secret, nil
}

// Get the CA bundle used by verrazzano ingress and the optional rancher-ca-additional secret
func (r *VerrazzanoManagedClusterReconciler) getAdminCaBundle() ([]byte, error) {
	var caBundle []byte

	// Append the CA bundle from verrazzano-tls-ca secret if it exists
	optSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.PrivateCABundle, false)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		// Combine the two CA bundles
		tlsCABundle := optSecret.Data[constants.RancherTLSCAKey]
		if len(tlsCABundle) > 0 {
			r.log.Debugf("Adding tls-ca bundle to Admin CA bundle")
			caBundle = tlsCABundle
		}
	}

	// Append the CA bundle from the ingress vzIngressSecret if it exists and is not already present in the bundle
	vzIngressSecret, err := r.getSecret(constants.VerrazzanoSystemNamespace, constants.VerrazzanoIngressTLSSecret, true)
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	tlsIngressCA, found := vzIngressSecret.Data[mcconstants.CaCrtKey]
	if found {
		if !strings.Contains(strings.TrimSpace(string(caBundle)), strings.TrimSpace(string(tlsIngressCA))) {
			r.log.Infof("Adding ingress CA cert bundle to Admin CA bundle")
			caBundle = append(caBundle, tlsIngressCA...)
		} else {
			r.log.Debugf("ingress CA bundle already present in bundle data, skipping")
		}
	}

	return caBundle, nil
}
