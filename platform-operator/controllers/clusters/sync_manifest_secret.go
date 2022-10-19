// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"context"
	"fmt"
	"strings"

	constants2 "github.com/verrazzano/verrazzano/pkg/mcconstants"
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

	caCertSecretKey = "cacrt"
)

// Create or update a secret that contains Kubernetes resource YAML which will be applied
// on the managed cluster to create resources needed there. The YAML has multiple Kubernetes
// resources separated by 3 hyphens ( --- ), so that applying the YAML will create/update multiple
// resources at once.  This YAML is stored in the Verrazzano manifest secret.
// This function returns a boolean and an error, where the boolean is true if the VMC was created by
// Verrazzano and the Rancher cluster ID has not yet been set in the status. This allows the
// calling function to requeue and process the VMC again before marking the VMC ready.
func (r *VerrazzanoManagedClusterReconciler) syncManifestSecret(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster) (bool, error) {
	// Builder used to build up the full YAML
	// For each secret, generate the YAML and append to the full YAML which contains multiple resources
	var sb = strings.Builder{}

	// add agent secret YAML
	agentYaml, err := r.getSecretAsYaml(GetAgentSecretName(vmc.Name), vmc.Namespace,
		constants.MCAgentSecret, constants.VerrazzanoSystemNamespace)
	if err != nil {
		return false, err
	}
	sb.Write([]byte(yamlSep))
	sb.Write(agentYaml)

	// add registration secret YAML
	regYaml, err := r.getSecretAsYaml(GetRegistrationSecretName(vmc.Name), vmc.Namespace,
		constants.MCRegistrationSecret, constants.VerrazzanoSystemNamespace)
	if err != nil {
		return false, err
	}
	sb.Write([]byte(yamlSep))
	sb.Write(regYaml)

	// if we created the VMC from a Rancher cluster, then wait for the cluster id to be populated in the status before we
	// attempt to fetch the registration manifest YAML
	vzVMCWaitingForClusterID := false

	if vmc.Labels != nil && vmc.Labels[createdByLabel] == createdByVerrazzano && len(vmc.Status.RancherRegistration.ClusterID) == 0 {
		r.log.Progressf("Waiting for Verrazzano-created VMC named %s to have a cluster id in the status before attempting to fetch Rancher registration manifest", vmc.Name)
		vzVMCWaitingForClusterID = true
	} else {
		// register the cluster with Rancher - the cluster will show as "pending" until the
		// Rancher YAML is applied on the managed cluster
		// NOTE: If this errors we log it and do not fail the reconcile
		var clusterID string
		rc, err := newRancherConfig(r.Client, r.log)
		if err != nil {
			msg := fmt.Sprintf("Failed to create Rancher API client: %v", err)
			r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationFailed, "", msg)
			r.log.Infof("Unable to connect to Rancher API on admin cluster, manifest secret will not contain Rancher YAML: %v", err)
		} else {
			var rancherYAML string
			rancherYAML, clusterID, err = registerManagedClusterWithRancher(rc, vmc.Name, vmc.Status.RancherRegistration.ClusterID, r.log)
			if err != nil {
				msg := fmt.Sprintf("Failed to register managed cluster with Rancher: %v", err)
				r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationFailed, clusterID, msg)
				r.log.Info("Failed to register managed cluster, manifest secret will not contain Rancher YAML")
			} else {
				msg := "Registration of managed cluster completed successfully"
				r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationCompleted, clusterID, msg)
				sb.WriteString(rancherYAML)
			}
		}
	}

	// create/update the manifest secret with the YAML
	_, err = r.createOrUpdateManifestSecret(vmc, sb.String())
	if err != nil {
		return false, err
	}

	// Save the ClusterRegistrationSecret name in the VMC
	vmc.Spec.ManagedClusterManifestSecret = GetManifestSecretName(vmc.Name)

	// finally, update the VMC
	err = r.Update(context.TODO(), vmc)
	if err != nil {
		return false, err
	}

	return vzVMCWaitingForClusterID, nil
}

// syncCACertSecret gets the CA cert from the managed cluster (if the cluster is active) and creates
// or updates the CA cert secret. If the secret is created, it also updates the VMC with the secret
// name. This function returns true if the sync was completed, false if it was not needed or not
// completed, and any error that occurred
func (r *VerrazzanoManagedClusterReconciler) syncCACertSecret(vmc *clusterapi.VerrazzanoManagedCluster) (bool, error) {
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		return false, nil
	}
	if len(vmc.Spec.CASecret) > 0 {
		return false, nil
	}
	rc, err := newRancherConfig(r.Client, r.log)
	if err != nil {
		return false, err
	}
	if rc == nil {
		return false, nil
	}

	isActive, err := isManagedClusterActiveInRancher(rc, clusterID, r.log)
	if err != nil {
		return false, err
	}
	if !isActive {
		r.log.Infof("Waiting for managed cluster with id %s to become active before fetching CA cert", clusterID)
		return false, nil
	}

	caCert, err := getCACertFromManagedCluster(rc, clusterID, r.log)
	if err != nil {
		return false, err
	}

	if len(caCert) > 0 {
		caSecretName := getCASecretName(vmc.Name)
		r.log.Infof("Retrieved CA cert from managed cluster with id %s, creating/updating secret %s", clusterID, caSecretName)
		if _, err := r.createOrUpdateCASecret(vmc, caCert); err != nil {
			return false, err
		}
		if len(caSecretName) > 0 {
			vmc.Spec.CASecret = caSecretName
			// update the VMC with ca secret name
			r.log.Infof("Updating VMC %s with managed cluster CA secret %s", vmc.Name, caSecretName)
			err = r.Update(context.TODO(), vmc)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

// Update the Rancher registration status
func (r *VerrazzanoManagedClusterReconciler) updateRancherStatus(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster, status clusterapi.RancherRegistrationStatus, rancherClusterID string, message string) {
	// Skip the update if the status has not changed
	if vmc.Status.RancherRegistration.Status == status &&
		vmc.Status.RancherRegistration.Message == message &&
		vmc.Status.RancherRegistration.ClusterID == rancherClusterID {
		return
	}
	vmc.Status.RancherRegistration.Status = status
	vmc.Status.RancherRegistration.ClusterID = rancherClusterID
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
		constants2.YamlKey: []byte(yamlData),
	}
	return nil
}

// createOrUpdateCASecret creates or updates the secret containing the managed cluster CA cert
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateCASecret(vmc *clusterapi.VerrazzanoManagedCluster, caCert string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = vmc.Namespace
	secret.Name = getCASecretName(vmc.Name)

	return controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		r.mutateCASecret(&secret, caCert)
		// This SetControllerReference call will trigger garbage collection i.e. the secret
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &secret, r.Scheme)
	})
}

// mutateCASecret mutates the CA secret, setting the CA cert data
func (r *VerrazzanoManagedClusterReconciler) mutateCASecret(secret *corev1.Secret, caCert string) error {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		caCertSecretKey: []byte(caCert),
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

// getCASecretName returns the CA secret name
func getCASecretName(vmcName string) string {
	return "ca-secret-" + vmcName
}
