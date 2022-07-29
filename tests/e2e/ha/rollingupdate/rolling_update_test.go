// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rollingupdate

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/ha"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

var clientset = k8sutil.GetKubernetesClientsetOrDie()
var t = framework.NewTestFramework("rolling_update")

var _ = t.Describe("Rolling Update", Label("f:platform-lcm:ha"), func() {
	t.It("does a rolling update of all nodes", func() {
		nodes, unschedulableNodes := getSchedulableAndUnschedulableNodes(clientset)
		// For each node pair, swap scheduling availability
		for i := range nodes {
			node := &nodes[i]
			unschedulableNode := &unschedulableNodes[i]
			// Mark node as unschedulable, mark unschedulableNode as schedulable
			ha.EventuallySetNodeScheduling(clientset, node.Name, true, t.Logs)
			ha.EventuallySetNodeScheduling(clientset, unschedulableNode.Name, false, t.Logs)
			// Evict all pods running on node
			ha.EventuallyEvictNode(clientset, node.Name, t.Logs)
			// Wait for pods on cluster to be ready before swapping the next node pair
			ha.EventuallyPodsReady(t.Logs, clientset)
			t.Logs.Infof("Finished rolling update from node[%s] to node[%s]", node.Name, unschedulableNode.Name)
		}
		// Create shutdown signal once rolling update is done
		ha.EventuallyCreateShutdownSignal(clientset, t.Logs)
	})
})

func getSchedulableAndUnschedulableNodes(cs *kubernetes.Clientset) ([]corev1.Node, []corev1.Node) {
	var nodes, taintedNodes []corev1.Node
	allNodes := ha.EventuallyGetNodes(cs, t.Logs)
	for _, node := range allNodes.Items {
		if !ha.IsControlPlaneNode(node) {
			if len(node.Spec.Taints) > 0 {
				taintedNodes = append(taintedNodes, node)
			} else {
				nodes = append(nodes, node)
			}
		}
	}
	Expect(len(nodes)).To(Equal(len(taintedNodes)))
	return nodes, taintedNodes
}
