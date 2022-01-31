// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"strings"

	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	yamlSep = "---\n"
)

// Create or update a secret that contains Kubernetes resource YAML which will be applied
// on the managed cluster to create resources needed there. The YAML has multiple Kubernetes
// resources separated by 3 hyphens ( --- ), so that applying the YAML will create/update multiple
// resources at once.  This YAML is stored in the Verrazzano manifest secret.
func (r *VerrazzanoManagedClusterReconciler) syncManifestSecret(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster) error {
	// Builder used to build up the full YAML
	// For each secret, generate the YAML and append to the full YAML which contains multiple resources
	var sb = strings.Builder{}

	// add agent secret YAML
	agentYaml, err := r.getSecretAsYaml(GetAgentSecretName(vmc.Name), vmc.Namespace,
		constants.MCAgentSecret, constants.VerrazzanoSystemNamespace)
	if err != nil {
		return err
	}
	sb.Write([]byte(yamlSep))
	sb.Write(agentYaml)

	// add registration secret YAML
	regYaml, err := r.getSecretAsYaml(GetRegistrationSecretName(vmc.Name), vmc.Namespace,
		constants.MCRegistrationSecret, constants.VerrazzanoSystemNamespace)
	if err != nil {
		return err
	}
	sb.Write([]byte(yamlSep))
	sb.Write(regYaml)

	// register the cluster with Rancher - the cluster will show as "pending" until the
	// Rancher YAML is applied on the managed cluster
	// NOTE: If this errors we log it and do not fail the reconcile
	if rancherYAML, err := registerManagedClusterWithRancher(r.Client, vmc.Name, r.log); err != nil {
		msg := fmt.Sprintf("Registration of managed cluster failed: %v", err)
		r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationFailed, msg)
		r.log.Info("Unable to register managed cluster with Rancher, manifest secret will not contain Rancher YAML")
	} else {
		msg := "Registration of managed cluster completed successfully"
		r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationCompleted, msg)
		sb.WriteString(rancherYAML)
	}

	// create/update the manifest secret with the YAML
	_, err = r.createOrUpdateManifestSecret(vmc, sb.String())
	if err != nil {
		return err
	}

	// Save the ClusterRegistrationSecret name in the VMC
	vmc.Spec.ManagedClusterManifestSecret = GetManifestSecretName(vmc.Name)
	err = r.Update(context.TODO(), vmc)
	if err != nil {
		return err
	}

	return nil
}

// Update the Rancher registration status
func (r *VerrazzanoManagedClusterReconciler) updateRancherStatus(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster, status clusterapi.RancherRegistrationStatus, message string) {
	// Skip the update if the status has not changed
	if vmc.Status.RancherRegistration.Status == status && vmc.Status.RancherRegistration.Message == message {
		return
	}
	vmc.Status.RancherRegistration.Status = status
	vmc.Status.RancherRegistration.Message = message
	err := r.Status().Update(ctx, vmc)
	if err != nil {
		r.log.Errorf("Failed to update Rancher registration status for VMC %s: %v", vmc.Name, err)
	}
}

// Create or update the manifest secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateManifestSecret(vmc *clusterapi.VerrazzanoManagedCluster, yamlData string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = vmc.Namespace
	secret.Name = GetManifestSecretName(vmc.Name)

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateManifestSecret(&secret, yamlData)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// Mutate the secret, setting the yaml data
func (r *VerrazzanoManagedClusterReconciler) mutateManifestSecret(secret *corev1.Secret, yamlData string) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		YamlKey: []byte(yamlData),
	}
	return nil
}

// Get the specified secret then convert to YAML.
func (r *VerrazzanoManagedClusterReconciler) getSecretAsYaml(name string, namespace string, targetName string, targetNamespace string) (yamlData []byte, err error) {
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		return []byte(""), fmt.Errorf("Failed to fetch the service account secret %s/%s, %v", namespace, name, err)
	}
	// Create a new ObjectMeta with target name and namespace
	secret.ObjectMeta = metav1.ObjectMeta{
		Namespace: targetNamespace,
		Name:      targetName,
	}
	yamlData, err = yaml.Marshal(secret)
	return yamlData, err
}

// GetManifestSecretName returns the manifest secret name
func GetManifestSecretName(vmcName string) string {
	const manifestSecretSuffix = "-manifest"
	return generateManagedResourceName(vmcName) + manifestSecretSuffix
}
