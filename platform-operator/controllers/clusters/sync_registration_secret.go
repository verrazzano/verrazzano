// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"

	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	k8net "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const vmiIngest = "vmi-system-es-ingest"

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
	fluentdESURL, fluentdESSecretName, err := r.getFluentdESURLSecret()
	if err != nil {
		return err
	}

	// Decide which ES URL to use.
	// If the fluentd ELASTICSEARCH_URL is the default "http://vmi-system-es-ingest-oidc:8775", use VMI ES ingress URL.
	// If the fluentd ELASTICSEARCH_URL is not the default, meaning it is a custom ES, use the external ES URL.
	esURL := fluentdESURL
	if esURL == "http://vmi-system-es-ingest-oidc:8775" {
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
	esCaBundle := adminCaBundle
	esSecretName := constants.VerrazzanoESInternal
	if fluentdESSecretName != "verrazzano" {
		esSecretName = fluentdESSecretName
		secretForCaBundle, err := r.getSecret(esSecretName)
		if err != nil {
			return err
		}
		esCaBundle = secretForCaBundle.Data["ca-bundle"]
	}
	esSecret, err := r.getSecret(esSecretName)
	if err != nil {
		return err
	}

	// Get the keycloak URL
	keycloakURL, err := r.getKeycloakURL()
	if err != nil {
		return err
	}

	// Build the secret data
	secret.Data = map[string][]byte{
		ManagedClusterNameKey: []byte(manageClusterName),
		ESURLKey:              []byte(esURL),
		ESCaBundleKey:         esCaBundle,
		UsernameKey:           esSecret.Data[UsernameKey],
		PasswordKey:           esSecret.Data[PasswordKey],
		KeycloakURLKey:        []byte(keycloakURL),
		AdminCaBundleKey:      adminCaBundle,
	}
	return nil
}

// GetRegistrationSecretName returns the registration secret name
func GetRegistrationSecretName(vmcName string) string {
	const registrationSecretSuffix = "-registration"
	return generateManagedResourceName(vmcName) + registrationSecretSuffix
}

func (r *VerrazzanoManagedClusterReconciler) getFluentdESURLSecret() (url string, secret string, err error) {
	// find the fluentd DaemonSet
	var daemonSet appsv1.DaemonSet
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      "fluentd",
	}
	if err := r.Get(context.TODO(), nsn, &daemonSet); err != nil {
		return "", "", fmt.Errorf("Failed to fetch the DaemonSet %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}

	// find the esURL from fluentd container env
	fluentdIndex := -1
	for i, container := range daemonSet.Spec.Template.Spec.Containers {
		if container.Name == "fluentd" {
			fluentdIndex = i
			break
		}
	}
	if fluentdIndex == -1 {
		return "", "", fmt.Errorf("fluentd container not found in fluentd daemonset: %s", daemonSet.Name)
	}
	esURL := ""
	for _, env := range daemonSet.Spec.Template.Spec.Containers[fluentdIndex].Env {
		if env.Name == constants.ElasticsearchURLEnvVar {
			esURL = env.Value
		}
	}

	// find the esSecret from secret-volume secretName
	esSecret := ""
	for _, volume := range daemonSet.Spec.Template.Spec.Volumes {
		if volume.Name == "secret-volume" && volume.Secret != nil {
			esSecret = volume.Secret.SecretName
		}
	}

	return esURL, esSecret, nil
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

// Get the system-tls secret
func (r *VerrazzanoManagedClusterReconciler) getAdminCaBundle() ([]byte, error) {
	var secret corev1.Secret
	nsn := types.NamespacedName{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.SystemTLS,
	}
	if err := r.Get(context.TODO(), nsn, &secret); err != nil {
		return nil, fmt.Errorf("Failed to fetch the secret %s/%s, %v", nsn.Namespace, nsn.Name, err)
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
