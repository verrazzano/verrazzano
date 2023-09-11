// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
)

const (
	rancherCcTenancyField     = "ocicredentialConfig-tenancyId"
	rancherCcUserField        = "ocicredentialConfig-userId"
	rancherCcFingerprintField = "ocicredentialConfig-fingerprint"
	rancherCcRegionField      = "ocicredentialConfig-region"
	rancherCcPassphraseField  = "ocicredentialConfig-passphrase" //nolint:gosec //#gosec G101
	rancherCcKeyField         = "ocicredentialConfig-privateKeyContents"
	ociCapiTenancyField       = "tenancy"
	ociCapiUserField          = "user"
	ociCapiFingerprintField   = "fingerprint"
	ociCapiRegionField        = "region"
	ociCapiPassphraseField    = "passphrase" //nolint:gosec //#gosec G101
	ociCapiKeyField           = "key"
)

type rancherMgmtCluster struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		GenericEngineConfig struct {
			CloudCredentialID string `json:"cloudCredentialId"`
		} `json:"genericEngineConfig"`
	} `json:"spec"`
}

// createOrUpdateCAPISecret updates CAPI secret based on the updated credentials
func (r *VerrazzanoSecretsReconciler) updateCAPISecret(updatedSecret *corev1.Secret, clusterCredential *corev1.Secret) error {
	data := map[string][]byte{
		ociCapiTenancyField:     updatedSecret.Data[rancherCcTenancyField],
		ociCapiUserField:        updatedSecret.Data[rancherCcUserField],
		ociCapiFingerprintField: updatedSecret.Data[rancherCcFingerprintField],
		ociCapiRegionField:      clusterCredential.Data[ociCapiRegionField],
		ociCapiPassphraseField:  updatedSecret.Data[rancherCcPassphraseField],
		ociCapiKeyField:         updatedSecret.Data[rancherCcKeyField],
	}
	clusterCredential.Data = data
	err := r.Client.Update(context.TODO(), clusterCredential)
	if err != nil {
		return err
	}
	return nil
}

// checkClusterCredentials checks whether the updated credential is being used by any OCNE cluster. If a cluster is using that credential, the OCNE cluster's copy of the credential also gets updated
func (r *VerrazzanoSecretsReconciler) checkClusterCredentials(updatedSecret *corev1.Secret) error {
	ocneClustersList, err := r.getOCNEClustersList()
	if err != nil {
		zap.S().Errorf("Failed to get OCNE Cluster list")
		return err
	}
	for _, cluster := range ocneClustersList.Items {
		var rancherMgmtCluster rancherMgmtCluster
		clusterJSON, err := cluster.MarshalJSON()
		if err != nil {
			return err
		}
		if err = json.Unmarshal(clusterJSON, &rancherMgmtCluster); err != nil {
			return err
		}
		// if the cluster is an OCNE cluster
		if rancherMgmtCluster.Spec.GenericEngineConfig.CloudCredentialID != "" {
			// extract cloud credential name from CloudCredentialID field
			capiCredential := strings.Split(rancherMgmtCluster.Spec.GenericEngineConfig.CloudCredentialID, ":")
			// if cloud credential name matches updatedSecret name, get and update the cc copy held by the cluster
			if len(capiCredential) >= 2 {
				if capiCredential[1] == updatedSecret.Name {
					secretName := fmt.Sprintf("%s-principal", rancherMgmtCluster.Metadata.Name)
					clusterCredential := &corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretName,
							Namespace: rancherMgmtCluster.Metadata.Name,
						},
					}
					// update cluster's cloud credential copy
					_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, clusterCredential, func() error {
						r.updateCAPISecret(updatedSecret, clusterCredential)
						return nil
					})
					if err != nil {
						zap.S().Errorf("Failed to update CAPI secret")
						return err
					}
				}
			}
		}
	}
	return nil
}

// getOCNEClustersList returns the list of OCNE clusters
func (r *VerrazzanoSecretsReconciler) getOCNEClustersList() (*unstructured.UnstructuredList, error) {
	var ocneClustersList *unstructured.UnstructuredList
	gvr := GetOCNEClusterAPIGVRForResource("clusters")
	ocneClustersList, err := r.DynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list %s/%s/%s: %v", gvr.Resource, gvr.Group, gvr.Version, err)
	}
	return ocneClustersList, nil
}

// GetOCNEClusterAPIGVRForResource returns a clusters.cluster.x-k8s.io GroupVersionResource structure
func GetOCNEClusterAPIGVRForResource(resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "management.cattle.io",
		Version:  "v3",
		Resource: resource,
	}
}

func isOCNECloudCredential(secret *corev1.Secret) bool {
	// if secret is a cloud credential in the cattle-global-data ns
	if secret.Namespace == rancher.CattleGlobalDataNamespace && secret.Data["ocicredentialConfig-fingerprint"] != nil {
		return true
	}
	// secret is not cloud credential
	return false
}
