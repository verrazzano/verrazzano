// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package node

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetK8sNodeList returns a list of Kubernetes nodes
func GetK8sNodeList(k8sClient client.Client) (*v1.NodeList, error) {
	nodeList := &v1.NodeList{}
	err := k8sClient.List(
		context.TODO(),
		nodeList,
		&client.ListOptions{},
	)
	return nodeList, err
}
