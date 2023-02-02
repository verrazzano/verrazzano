// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/dynamic"
)

var (
	specField      = "spec"
	specReplicas   = []string{specField, "replicas"}
	statusClusters = []string{"status", "clusters"}
)

// getScheme returns the WebLogic scheme needed to get unstructured data
func getScheme() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "weblogic.oracle",
		Version:  "v8",
		Resource: "domains",
	}
}

// GetReadyReplicas returns the readyReplicas from the first cluster in domain status
func GetReadyReplicas(client dynamic.Interface, namespace string, name string) (int64, error) {
	domain, err := client.Resource(getScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	clusters, found, err := unstructured.NestedSlice(domain.Object, statusClusters...)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, fmt.Errorf("Failed to get clusters %v", err)
	}
        readyReplicas := clusters[0].(map[string]interface{})["readyReplicas"]
        if readyReplicas != nil {
            return readyReplicas.(int64), nil
        }
        return 0, nil
}

// GetCurrentReplicas returns the replicas value from /spec/replicas
func GetCurrentReplicas(client dynamic.Interface, namespace string, name string) (int64, error) {
	specReplicas = []string{specField, "replicas"}
	domain, err := client.Resource(getScheme()).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	rep, found, err := unstructured.NestedInt64(domain.Object, specReplicas...)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, fmt.Errorf("Failed to get replicas %v %v", specReplicas, err)
	}
	return rep, nil

}

// PatchReplicas patches the replicas at /spec/replicas
func PatchReplicas(client dynamic.Interface, namespace string, name string, replicas int64) error {
	// patchInt64Value specifies a patch operation for a int64.
	type patchInt64Value struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value int64  `json:"value"`
	}
	payload := []patchInt64Value{{
		Op:    "replace",
		Path:  "/spec/replicas",
		Value: replicas,
	}}
	payloadBytes, _ := json.Marshal(payload)
	_, err := client.Resource(getScheme()).Namespace(namespace).Patch(context.TODO(), name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
