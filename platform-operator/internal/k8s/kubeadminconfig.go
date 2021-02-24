package k8s

import (
	"context"
	"fmt"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterStatus contains the kubeadmin config ApiEndpoint map
type ClusterStatus struct {
	ApiEndpoints map[string]ApiEndpoint `json:"apiEndpoints"`
}

// ApiEndpoint contains the kubeadmin config information needed to access the API server
type ApiEndpoint struct {
	AdvertiseAddress string `json:"advertiseAddress"`
	BindPort         string `json:"bindPort"`
}

// GetApiServerURL gets the external host:port of the API server
func GetApiServerURL(client clipkg.Client) (string, error) {
	const (
		kubeSystem       = "kube-system"
		kubeAdminConfig  = "kubeadm-config"
		clusterStatusKey = "ClusterStatus"
		apiEndpointsKey  = "apiEndpoints"
	)

	// Get the service account token from the secret
	var cm corev1.ConfigMap
	nsn := types.NamespacedName{
		Namespace: kubeSystem,
		Name:      kubeAdminConfig,
	}
	if err := client.Get(context.TODO(), nsn, &cm); err != nil {
		return "", fmt.Errorf("Failed to fetch the kube adimin configmap %s/%s, %v", nsn.Namespace, nsn.Name, err)
	}

	statusData := cm.Data[clusterStatusKey]
	if len(statusData) == 0 {
		return "", fmt.Errorf("Missing ClusterStatus in the configmap %s/%s", kubeSystem, kubeAdminConfig)
	}

	// Convert the data into a cluster status
	var cs ClusterStatus
	err := yaml.Unmarshal([]byte(statusData), &cs)
	if err != nil {
		return "", err
	}
	for _, ep := range cs.ApiEndpoints {
		return fmt.Sprintf("https://%s:%v", ep.AdvertiseAddress, ep.BindPort), nil
	}
	return "", fmt.Errorf("Missing ClusterStatus ApiEndpoints in the configmap %s/%s", kubeSystem, kubeAdminConfig)
}
