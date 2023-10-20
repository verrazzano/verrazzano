// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"fmt"
	internalcapi "github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	constants3 "github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	clusterapi "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/rancher"
	constants2 "github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (r *VerrazzanoManagedClusterReconciler) syncManifestSecret(ctx context.Context, rancherEnabled bool, vmc *clusterapi.VerrazzanoManagedCluster) (bool, error) {
	// Builder used to build up the full YAML
	// For each secret, generate the YAML and append to the full YAML which contains multiple resources
	var sb = strings.Builder{}

	// if we created the VMC from a Rancher cluster, then wait for the cluster id to be populated in the status before we
	// attempt to fetch the registration manifest YAML
	vzVMCWaitingForClusterID := false

	clusterID := vmc.Status.RancherRegistration.ClusterID
	var rc *rancherutil.RancherConfig
	if rancherEnabled {
		if vmc.Labels != nil && vmc.Labels[rancher.CreatedByLabel] == rancher.CreatedByVerrazzano && len(clusterID) == 0 {
			r.log.Progressf("Waiting for Verrazzano-created VMC named %s to have a cluster id in the status before attempting to fetch Rancher registration manifest", vmc.Name)
			vzVMCWaitingForClusterID = true
		} else {
			var err error
			// register the cluster with Rancher - the cluster will show as "pending" until the
			// Rancher YAML is applied on the managed cluster
			// NOTE: If this errors we log it and do not fail the reconcile
			rc, err = rancherutil.NewAdminRancherConfig(r.Client, r.RancherIngressHost, r.log)
			if err != nil {
				msg := "Failed to create Rancher API client"
				r.log.Infof("Unable to connect to Rancher API on admin cluster, manifest secret will not contain Rancher YAML: %v", err)
				r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationFailed, "", msg)
			} else {
				var rancherYAML string
				rancherYAML, clusterID, err = RegisterManagedClusterWithRancher(rc, vmc.Name, vmc.Status.RancherRegistration.ClusterID, r.log)
				if err != nil {
					msg := "Failed to register managed cluster with Rancher"
					// Even if there was a failure, if the cluster id was retrieved and is currently empty
					// on the VMC, populate it during status update
					r.log.Info("Failed to register managed cluster, manifest secret will not contain Rancher YAML")
					r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationFailed, clusterID, msg)
				} else if len(rancherYAML) == 0 {
					// we successfully called the Rancher API but for some reason the returned registration manifest YAML is empty,
					// set the status on the VMC and return an error so we reconcile again
					r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationFailed, clusterID, "Empty Rancher manifest YAML")
					msg := fmt.Sprintf("Failed retrieving Rancher manifest, YAML is an empty string for cluster ID %s", clusterID)
					r.log.Infof(msg)
					return vzVMCWaitingForClusterID, r.log.ErrorNewErr(msg)
				} else {
					msg := fmt.Sprintf("Registration of managed cluster completed successfully for cluster %s with ID %s", vmc.Name, clusterID)
					r.log.Once(msg)
					if vmc.Status.RancherRegistration.Status != clusterapi.RegistrationApplied {
						r.updateRancherStatus(ctx, vmc, clusterapi.RegistrationCompleted, clusterID, msg)
					}
					sb.WriteString(rancherYAML)
				}
			}
		}
	}

	// if registration was successful and there is a cluster ID OR there is CAPI access to cluster...
	isActive := false
	if len(clusterID) > 0 {
		isActive, _ = isManagedClusterActiveInRancher(rc, clusterID, r.log)
	}
	if vmc.Status.ClusterRef != nil || (len(clusterID) > 0 && isActive) {
		// check for existence of verrazzano-system namespace
		vsNamespaceExists, _ := isNamespaceCreated(vmc, r, clusterID, constants.VerrazzanoSystemNamespace)

		if vsNamespaceExists {
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
		}
	}
	// create/update the manifest secret with the YAML
	_, err := r.createOrUpdateManifestSecret(vmc, sb.String())
	if err != nil {
		return vzVMCWaitingForClusterID, err
	}

	// finally, update the VMC
	existingVMC := &clusterapi.VerrazzanoManagedCluster{}
	err = r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Namespace, Name: vmc.Name}, existingVMC)
	if err != nil {
		r.log.Errorf("Failed to get the existing VMC %s from the cluster: %v", vmc.Name, err)
	}
	existingVMC.Spec.ManagedClusterManifestSecret = GetManifestSecretName(vmc.Name)

	err = r.Update(ctx, existingVMC)
	if err != nil {
		return vzVMCWaitingForClusterID, err
	}

	return vzVMCWaitingForClusterID, nil
}

// syncCACertSecret gets the CA cert from the managed cluster (if the cluster is active) and creates
// or updates the CA cert secret. If the secret is created, it also updates the VMC with the secret
// name. This function returns true if the sync was completed, false if it was not needed or not
// completed, and any error that occurred
func (r *VerrazzanoManagedClusterReconciler) syncCACertSecret(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster, rancherEnabled bool) (bool, error) {
	caCert := ""
	var err error
	if rancherEnabled {
		caCert, err = r.getCACertViaRancher(vmc)
		if err != nil {
			return false, err
		}
	}
	if len(caCert) == 0 && vmc.Status.ClusterRef != nil {
		caCert, err = r.getCACertViaCAPI(ctx, vmc)
		if err != nil {
			return false, err
		}
	}

	if len(caCert) > 0 {
		caSecretName := getCASecretName(vmc.Name)
		r.log.Infof("Retrieved CA cert from managed cluster, creating/updating secret %s", caSecretName)
		if _, err = r.createOrUpdateCASecret(vmc, caCert); err != nil {
			return false, err
		}
		if len(caSecretName) > 0 {
			vmc.Spec.CASecret = caSecretName
			// update the VMC with ca secret name
			r.log.Infof("Updating VMC %s with managed cluster CA secret %s", vmc.Name, caSecretName)
			// Replace the VMC in the update call with a copy
			// That way the existing VMC status updates do not get overwritten by the update
			updateVMC := vmc.DeepCopy()
			err = r.Update(context.TODO(), updateVMC)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

	return false, nil
}

func (r *VerrazzanoManagedClusterReconciler) getCACertViaCAPI(ctx context.Context, vmc *clusterapi.VerrazzanoManagedCluster) (string, error) {
	caCrt := ""
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(internalcapi.GVKCAPICluster)
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Status.ClusterRef.Namespace, Name: vmc.Status.ClusterRef.Name}, cluster)
	if err != nil && !errors.IsNotFound(err) {
		return "", err
	}
	// register the cluster if Verrazzano installed on workload cluster
	workloadClient, err := r.getWorkloadClusterClient(cluster)
	if err != nil {
		r.log.Errorf("Error getting workload cluster %s client: %v", cluster.GetName(), err)
		return "", err
	}

	// ensure Verrazzano is installed and ready in workload cluster
	ready, err := r.isVerrazzanoReady(ctx, workloadClient)
	if err != nil {
		return "", err
	}
	if !ready {
		r.log.Progressf("Verrazzano not installed or not ready in cluster %s", cluster.GetName())
		return "", err
	}
	// if verrazzano-tls-ca exists, the cluster is untrusted
	err = workloadClient.Get(ctx, types.NamespacedName{Name: constants3.PrivateCABundle,
		Namespace: constants.VerrazzanoSystemNamespace}, &corev1.Secret{})
	if err != nil {
		if errors.IsNotFound(err) {
			// cluster is trusted
			r.log.Infof("Cluster %s is using trusted certs", cluster.GetName())
		} else {
			// unexpected error
			return "", err
		}
	} else {
		// need to create a CA secret in admin cluster
		// get the workload cluster API CA cert
		caCrt, err = r.getWorkloadClusterCACert(workloadClient)
		if err != nil {
			return "", err
		}
		// persist the workload API certificate on the admin cluster
		adminWorkloadCertSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ca-secret-%s", cluster.GetName()),
			Namespace: constants.VerrazzanoMultiClusterNamespace}}
		if _, err := ctrl.CreateOrUpdate(context.TODO(), r.Client, adminWorkloadCertSecret, func() error {
			if len(adminWorkloadCertSecret.Data) == 0 {
				adminWorkloadCertSecret.Data = make(map[string][]byte)
			}
			adminWorkloadCertSecret.Data["cacrt"] = []byte(caCrt)

			return nil
		}); err != nil {
			return "", err
		}
	}

	return caCrt, nil
}

func (r *VerrazzanoManagedClusterReconciler) getCACertViaRancher(vmc *clusterapi.VerrazzanoManagedCluster) (string, error) {
	clusterID := vmc.Status.RancherRegistration.ClusterID
	if len(clusterID) == 0 {
		return "", nil
	}
	if len(vmc.Spec.CASecret) > 0 {
		return "", nil
	}
	rc, err := rancherutil.NewAdminRancherConfig(r.Client, r.RancherIngressHost, r.log)
	if err != nil {
		return "", err
	}
	if rc == nil {
		return "", nil
	}

	isActive, err := isManagedClusterActiveInRancher(rc, clusterID, r.log)
	if err != nil {
		return "", err
	}
	if !isActive {
		r.log.Progressf("Waiting for managed cluster with id %s to become active before fetching CA cert", clusterID)
		return "", nil
	}

	caCert, err := getCACertFromManagedCluster(rc, clusterID, r.log)
	if err != nil {
		return "", err
	}

	return caCert, nil
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
	// don't wipe out existing cluster id with empty string
	if rancherClusterID != "" {
		vmc.Status.RancherRegistration.ClusterID = rancherClusterID
	}
	vmc.Status.RancherRegistration.Message = message

	// Fetch the existing VMC to avoid conflicts in the status update
	existingVMC := &clusterapi.VerrazzanoManagedCluster{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: vmc.Namespace, Name: vmc.Name}, existingVMC)
	if err != nil {
		r.log.Errorf("Failed to get the existing VMC %s from the cluster: %v", vmc.Name, err)
	}
	existingVMC.Status.RancherRegistration = vmc.Status.RancherRegistration

	err = r.Status().Update(ctx, existingVMC)
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
func (r *VerrazzanoManagedClusterReconciler) mutateManifestSecret(secret *corev1.Secret, yamlData string) {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		constants2.YamlKey: []byte(yamlData),
	}
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
func (r *VerrazzanoManagedClusterReconciler) mutateCASecret(secret *corev1.Secret, caCert string) {
	secret.Type = corev1.SecretTypeOpaque
	secret.Data = map[string][]byte{
		caCertSecretKey: []byte(caCert),
	}
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

// isVerrazzanoReady checks to see whether the Verrazzano resource on the workload cluster is ready
func (r *VerrazzanoManagedClusterReconciler) isVerrazzanoReady(ctx context.Context, workloadClient client.Client) (bool, error) {
	if err := workloadClient.Get(ctx,
		types.NamespacedName{Name: constants.Verrazzano, Namespace: constants.VerrazzanoSystemNamespace},
		&corev1.Secret{}); err != nil {
		if !errors.IsNotFound(err) {
			r.log.Debugf("Failed to retrieve verrazzano secret: %v", err)
			return false, err
		}
		r.log.Debugf("Verrazzano secret not found")
		return false, nil
	}
	return true, nil
}

// getWorkloadClusterCACert retrieves the API endpoint CA certificate from the workload cluster
func (r *VerrazzanoManagedClusterReconciler) getWorkloadClusterCACert(workloadClient client.Client) (string, error) {
	caCrtSecret := &corev1.Secret{}
	err := workloadClient.Get(context.TODO(), types.NamespacedName{
		Name:      constants3.VerrazzanoIngressTLSSecret,
		Namespace: constants.VerrazzanoSystemNamespace},
		caCrtSecret)
	if err != nil {
		return "", err
	}
	caCrt, ok := caCrtSecret.Data["ca.crt"]
	if !ok {
		return "", fmt.Errorf("Workload cluster CA certificate not found in verrazzano-tls secret")
	}
	return string(caCrt), nil
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
