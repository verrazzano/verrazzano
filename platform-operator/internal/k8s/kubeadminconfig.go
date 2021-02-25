// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package k8s

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStatus contains ApiEndpoint map stored in the kubeadmin config map
type ClusterStatus struct {
	ApiEndpoints map[string]ApiEndpoint `json:"apiEndpoints"`
}

// ApiEndpoint contains the kubeadmin config information needed to access the API server
type ApiEndpoint struct {
	AdvertiseAddress string `json:"advertiseAddress"`
	BindPort         string `json:"bindPort"`
}

const (
	KubeSystem       = "kube-system"
	KubeAdminConfig  = "kubeadm-config"
	ClusterStatusKey = "ClusterStatus"
)

// GetApiServerURL gets the external hURL of the API server
func GetApiServerURL(client clipkg.Client) (string, error) {
	// Get the configmap which has the info needed to build the URL
	var cm corev1.ConfigMap
	nsn := types.NamespacedName{
		Namespace: KubeSystem,
		Name:      KubeAdminConfig,
	}
	if err := client.Get(context.TODO(), nsn, &cm); err != nil {
		return "", fmt.Errorf("Failed to fetch the kube adimin configmap %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}
	statusData := cm.Data[ClusterStatusKey]
	if len(statusData) == 0 {
		return "", fmt.Errorf("Missing ClusterStatus in the configmap %s/%s", KubeSystem, KubeAdminConfig)
	}

	// Unmarshal the data then build the URL
	var cs ClusterStatus
	err := yaml.Unmarshal([]byte(statusData), &cs)
	if err != nil {
		return "", err
	}
	for _, ep := range cs.ApiEndpoints {
		return fmt.Sprintf("https://%s:%v", ep.AdvertiseAddress, ep.BindPort), nil
	}
	return "", fmt.Errorf("Missing ClusterStatus ApiEndpoints in the configmap %s/%s", KubeSystem, KubeAdminConfig)
}
