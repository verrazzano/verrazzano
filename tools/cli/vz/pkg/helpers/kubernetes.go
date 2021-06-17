package helpers

import (
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

//go:generate mockgen -source kubernetes.go -destination ../../mock/kubernetes_mock.go -package mock

type Kubernetes interface {
	NewClientSet() (clientset.Interface, error)
	NewKubernetesClientSet() kubernetes.Interface
}
