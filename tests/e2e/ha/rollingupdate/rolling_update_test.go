// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rollingupdate

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/ha"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var clientset = k8sutil.GetKubernetesClientsetOrDie()
var t = framework.NewTestFramework("rolling_update")

var _ = t.Describe("Rolling Update", Label("f:platform-lcm:ha"), func() {
	t.It("does a rolling update of all nodes", func() {
		nodes, unschedulableNodes := getNodes(clientset)
		for i := range nodes {
			node := &nodes[i]
			unschedulableNode := &unschedulableNodes[i]
			// Mark node as unschedulable, mark unschedulableNode as schedulable
			swapScheduling(clientset, node, unschedulableNode)
			// Delete all pods running on node
			deletePodsForNode(clientset, node)
			// Wait for pods on cluster to be ready
			eventuallyPodsReady(clientset)
			t.Logs.Infof("Finished rolling update from node %s to %s", node.Name, unschedulableNode.Name)
		}
	})
})

func swapScheduling(cs *kubernetes.Clientset, node, unschedulableNode *corev1.Node) {
	Eventually(func() bool {
		if err := swapNodeScheduling(cs, node.Name); err != nil {
			t.Logs.Errorf("Failed to mark node %s as unschedulable: %v", node.Name, err)
			return false
		}
		if err := swapNodeScheduling(cs, unschedulableNode.Name); err != nil {
			t.Logs.Errorf("Failed to mark node %s as schedulable: %v", unschedulableNode.Name, err)
			return false
		}
		return true
	}, ha.WaitTimeout, ha.PollingInterval).Should(BeTrue())
}

func swapNodeScheduling(cs *kubernetes.Clientset, name string) error {
	node, err := cs.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	node.Spec.Unschedulable = !node.Spec.Unschedulable
	if _, err := cs.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

func deletePodsForNode(cs *kubernetes.Clientset, node *corev1.Node) {
	Eventually(func() bool {
		pods, err := cs.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
		})
		if err != nil {
			t.Logs.Errorf("Failed to get pods for node %s: %v", node.Name, err)
			return false
		}

		for i := range pods.Items {
			pod := &pods.Items[i]
			if err := cs.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
				t.Logs.Errorf("Failed to delete pod %s for node %s: %v", pod.Name, node.Name, err)
			}
		}
		return true
	}, ha.WaitTimeout, ha.PollingInterval).Should(BeTrue())
}

func getNodes(cs *kubernetes.Clientset) ([]corev1.Node, []corev1.Node) {
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

func eventuallyPodsReady(cs *kubernetes.Clientset) {
	var pods *corev1.PodList
	Eventually(func() bool {
		var err error
		pods, err = cs.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Logs.Info("Failed to get pods: %v", err)
			return false
		}

		for _, pod := range pods.Items {
			if !isPodReadyOrCompleted(pod) {
				return false
			}
		}
		return true

	}, ha.WaitTimeout, ha.PollingInterval).Should(BeTrue())
}

func isPodReadyOrCompleted(pod corev1.Pod) bool {
	switch pod.Status.Phase {
	case corev1.PodSucceeded:
		return true
	case corev1.PodRunning:
		for _, c := range pod.Status.ContainerStatuses {
			if !c.Ready {
				return false
			}
		}
		return true
	default:
		return false
	}
}
