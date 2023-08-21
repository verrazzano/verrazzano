// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	ociTenancyField              = "ocicredentialConfig-tenancyId"
	ociUserField                 = "ocicredentialConfig-userId"
	ociFingerprintField          = "ocicredentialConfig-fingerprint"
	ociRegionField               = "ocicredentialConfig-region"
	ociPassphraseField           = "ocicredentialConfig-passphrase" //nolint:gosec //#gosec G101
	ociKeyField                  = "ocicredentialConfig-privateKeyContents"
	ociUseInstancePrincipalField = "useInstancePrincipal"
)

// Struct to unmarshall the ocne clusters list
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

func (r *VerrazzanoSecretsReconciler) watchCloudCredForUpdate(updatedSecret *corev1.Secret) error {
	if isOCNECloudCredential(updatedSecret) {
		//get ocne clusters and then update copy of cloud credential if necessary
		dynClient, err := getDynamicClient()
		if err != nil {
			return err
		}
		if err = r.updateOCNEclusterCloudCreds(updatedSecret, dynClient); err != nil {
			return err
		}
	}
	return nil
}

// createOrUpdateCAPISecret updates CAPI based on the updated credentials
func (r *VerrazzanoSecretsReconciler) updateCAPISecret(updatedSecret *corev1.Secret, clusterCredential *corev1.Secret) error {
	data := map[string][]byte{
		ociTenancyField:              updatedSecret.Data[ociTenancyField],
		ociUserField:                 updatedSecret.Data[ociUserField],
		ociFingerprintField:          updatedSecret.Data[ociFingerprintField],
		ociRegionField:               clusterCredential.Data[ociRegionField],
		ociPassphraseField:           updatedSecret.Data[ociPassphraseField],
		ociKeyField:                  updatedSecret.Data[ociKeyField],
		ociUseInstancePrincipalField: updatedSecret.Data[ociUseInstancePrincipalField],
	}
	clusterCredential.Data = data
	err := r.Client.Update(context.TODO(), clusterCredential)
	if err != nil {
		return err
	}
	return nil
}

func (r *VerrazzanoSecretsReconciler) updateOCNEclusterCloudCreds(updatedSecret *corev1.Secret, dynClient dynamic.Interface) error {
	ocneClustersList, err := getOCNEClustersList(dynClient)
	if err != nil {
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
			cloudCredentialStringSplit := strings.Split(rancherMgmtCluster.Spec.GenericEngineConfig.CloudCredentialID, ":")
			// if cloud credential name matches updatedSecret name, get and update the cc copy held by the cluster
			if cloudCredentialStringSplit[1] == updatedSecret.Name {
				secretName := fmt.Sprintf("%s-principal", rancherMgmtCluster.Metadata.Name)
				clusterCredential := &corev1.Secret{}
				if err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: rancherMgmtCluster.Metadata.Name, Name: secretName}, clusterCredential); err != nil {
					return err
				}
				// update cluster's cloud credential copy
				if err = r.updateCAPISecret(updatedSecret, clusterCredential); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// getOCNEClustersList returns the list of OCNE clusters
func getOCNEClustersList(dynClient dynamic.Interface) (*unstructured.UnstructuredList, error) {
	var ocneClustersList *unstructured.UnstructuredList
	gvr := GetOCNEClusterAPIGVRForResource("clusters")
	ocneClustersList, err := dynClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
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

// GetDynamicClient returns a dynamic client needed to access Unstructured data
func getDynamicClient() (dynamic.Interface, error) {
	config, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}

func isOCNECloudCredential(secret *corev1.Secret) bool {
	// if secret is a cloud credential in the cattle-global-data ns
	if secret.Namespace == rancher.CattleGlobalDataNamespace && secret.Data["ocicredentialConfig-fingerprint"] != nil {
		return true
	}
	// secret is not cloud credential
	return false
}
