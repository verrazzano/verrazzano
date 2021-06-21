// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	clientset "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

type Kubernetes interface {
	NewClientSet() (clientset.Interface, error)
	NewKubernetesClientSet() kubernetes.Interface
}
