// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package node

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	controlPlaneTaint = "node-role.kubernetes.io/control-plane"
	masterTaint       = "node-role.kubernetes.io/master"
	loadBalancerTaint = "node.kubernetes.io/exclude-from-external-load-balancers"
)

// GetK8sNodeList returns a list of Kubernetes nodes.
func GetK8sNodeList(k8sClient client.Client) (*v1.NodeList, error) {
	nodeList := &v1.NodeList{}
	err := k8sClient.List(
		context.TODO(),
		nodeList,
		&client.ListOptions{},
	)
	return nodeList, err
}

// SetControlPlaneScheduling will mark control plane nodes for scheduling by removing taints.
func SetControlPlaneScheduling(ctx context.Context, k8sClient client.Client) error {
	nodes, err := GetK8sNodeList(k8sClient)
	if err != nil {
		return err
	}
	for i := range nodes.Items {
		node := &nodes.Items[i]
		var taints []v1.Taint
		for _, taint := range node.Spec.Taints {
			if !isControlPlaneNoScheduleTaint(taint) {
				taints = append(taints, taint)
			}
		}
		node.Spec.Taints = taints
		delete(node.Labels, loadBalancerTaint)
		if err := k8sClient.Update(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func isControlPlaneNoScheduleTaint(taint v1.Taint) bool {
	return taint.Key == controlPlaneTaint || taint.Key == masterTaint
}
