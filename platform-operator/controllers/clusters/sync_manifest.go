// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"
	clusterapi "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	manifestSecretSuffix = "-manifest"
	yamlSep              = "---\n"
	yamlKey              = "yaml"
)

// Create or update a secret that contains Kubernetes resource YAML which will be applied
// on the managed cluster to create resources needed there. The YAML has multiple Kubernetes
// resources separated by 3 hyphens ( --- ), so that applying the YAML will create/update multiple
// resources at once.  This YAML is stored in the Verrazzano manifest secret.
func (r *VerrazzanoManagedClusterReconciler) syncManifestSecret(vmc *clusterapi.VerrazzanoManagedCluster) error {
	// Builder used to build up the full YAML
	var sb = strings.Builder{}

	// generate YAML for each resource, then combine to a single YAML
	y, err := r.getSecretAsYaml(getRegistrationSecretName(vmc.Name), vmc.Namespace)
	if err != nil {
		return err
	}
	sb.Write([]byte(yamlSep))
	sb.Write(y)

	// create/update the manifest secret with the YAML
	_, err = r.createOrUpdateManifestSecret(vmc, sb.String())
	if err != nil {
		return err
	}

	// Save the ClusterRegistrationSecret name in the VMC
	vmc.Spec.ManagedClusterManifestSecret = getManifestSecretName(vmc.Name)
	err = r.Update(context.TODO(), vmc)
	if err != nil {
		return err
	}

	return nil
}

// Create or update the manifest secret
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateManifestSecret(vmc *clusterapi.VerrazzanoManagedCluster, yamlData string) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = vmc.Namespace
	secret.Name = getManifestSecretName(vmc.Name)

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
		yamlKey: []byte(yamlData),
	}
	return nil
}

// Get the specified secre then convert to YAML.
func (r *VerrazzanoManagedClusterReconciler) getSecretAsYaml(name string, namespace string) (yamlData []byte, err error) {
	var secret corev1.Secret
	secretNsn := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := r.Get(context.TODO(), secretNsn, &secret); err != nil {
		return []byte(""), fmt.Errorf("Failed to fetch the service account secret %s/%s, %v", namespace, name, err)
	}
	// Create a new ObjectMeta with just namespace and name
	secret.ObjectMeta = metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
	}
	yamlData, err = yaml.Marshal(secret)
	return yamlData, err
}

func getManifestSecretName(vmcName string) string {
	return generateManagedResourceName(vmcName) + manifestSecretSuffix
}
