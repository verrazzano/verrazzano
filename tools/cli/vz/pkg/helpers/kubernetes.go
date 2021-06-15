package helpers

import (
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	restclient "k8s.io/client-go/rest"
)

//go:generate mockgen -source kubernetes.go -destination ../../mock/kubernetes_mock.go -package mock

type Kubernetes interface {
	GetKubeConfig() *restclient.Config
	NewClientSet() (clientset.Interface, error)
}
