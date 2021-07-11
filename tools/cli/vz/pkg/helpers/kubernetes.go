// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	projectclientset "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned"
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	verrazzanoclientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Kubernetes interface {
	GetKubeConfig() *rest.Config
	NewClustersClientSet() (clientset.Interface, error)
	NewProjectClientSet() (projectclientset.Interface, error)
	NewVerrazzanoClientSet() (verrazzanoclientset.Interface, error)
	NewClientSet() kubernetes.Interface
}
