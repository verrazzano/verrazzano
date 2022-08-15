// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scheduling

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/ha"
	corev1 "k8s.io/api/core/v1"
)

var t = framework.NewTestFramework("scheduling")
var clientset = k8sutil.GetKubernetesClientsetOrDie()

var _ = t.Describe("OKE Scheduling", Label("f:platform-lcm:ha"), func() {
	t.It("marks half the worker nodes in the cluster as unschedulable", func() {
		nodes := ha.EventuallyGetNodes(clientset, t.Logs)
		var workerNodes []corev1.Node
		for _, node := range nodes.Items {
			// Exclude control plane node from tests, only process worker nodes
			if !ha.IsControlPlaneNode(node) {
				workerNodes = append(workerNodes, node)
			}
		}
		// Mark half the worker nodes as unschedulable, and evict their pods for rescheduling
		for i := 0; i < len(workerNodes)/2; i++ {
			workerNode := &workerNodes[i]
			// Set the node to be unschedulable
			ha.EventuallySetNodeScheduling(clientset, workerNode.Name, true, t.Logs)
			// Evict pods from node
			ha.EventuallyEvictNode(clientset, workerNode.Name, t.Logs)
		}
		// Wait for pods to be ready after rescheduling
		ha.EventuallyPodsReady(t.Logs, clientset)
	})
})
