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
type ocneCluster struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		GenericEngineConfig struct {
			CloudCredentialID string `json:"cloudCredentialId"`
		} `json:"genericEngineConfig"`
	} `json:"spec"`
}

func watchCloudCredForUpdate(r *VerrazzanoSecretsReconciler, updatedSecret *corev1.Secret) error {
	if r.isOCNECloudCredential(updatedSecret) {
		//get ocne clusters and then update copy of cloud credential if necessary
		dynClient, err := getDynamicClient()
		if err != nil {
			return err
		}
		if err = updateOCNEclusterCloudCreds(updatedSecret, r, dynClient); err != nil {
			return err
		}
	}
	return nil
}

// createOrUpdateCAPISecret updates CAPI based on the updated credentials
func updateCAPISecret(r *VerrazzanoSecretsReconciler, updatedSecret *corev1.Secret, clusterCredential *corev1.Secret) error {
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

func updateOCNEclusterCloudCreds(updatedSecret *corev1.Secret, r *VerrazzanoSecretsReconciler, dynClient dynamic.Interface) error {
	ocneClustersList, err := getOCNEClustersList(dynClient)
	if err != nil {
		return err
	}
	for _, cluster := range ocneClustersList.Items {
		var ocneStruct ocneCluster
		clusterJSON, err := cluster.MarshalJSON()
		if err != nil {
			return err
		}
		if err = json.Unmarshal(clusterJSON, &ocneStruct); err != nil {
			return err
		}
		// if the cluster is an OCNE cluster
		if ocneStruct.Spec.GenericEngineConfig.CloudCredentialID != "" {
			// extract cloud credential name from CloudCredentialID field
			cloudCredentialStringSplit := strings.Split(ocneStruct.Spec.GenericEngineConfig.CloudCredentialID, ":")
			// if cloud credential name matches updatedSecret name, get and update the cc copy held by the cluster
			if cloudCredentialStringSplit[1] == updatedSecret.Name {
				secretName := fmt.Sprintf("%s-principal", ocneStruct.Metadata.Name)
				clusterCredential := &corev1.Secret{}
				if err = r.Client.Get(context.TODO(), client.ObjectKey{Namespace: ocneStruct.Metadata.Name, Name: secretName}, clusterCredential); err != nil {
					return err
				}
				// update cluster's cloud credential copy
				if err = updateCAPISecret(r, updatedSecret, clusterCredential); err != nil {
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

func (r *VerrazzanoSecretsReconciler) isOCNECloudCredential(secret *corev1.Secret) bool {
	// if secret is a cloud credential in the cattle-global-data ns
	if secret.Namespace == rancher.CattleGlobalDataNamespace && secret.Data["ocicredentialConfig-fingerprint"] != nil {
		return true
	}
	// secret is not cloud credential
	return false
}
