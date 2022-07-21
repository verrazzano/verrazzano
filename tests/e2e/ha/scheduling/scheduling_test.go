// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scheduling

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/ha"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var t = framework.NewTestFramework("scheduling")
var clientset = k8sutil.GetKubernetesClientsetOrDie()

var _ = t.Describe("Kind Scheduling", Label("f:platform-lcm:ha"), func() {
	t.It("marks half the worker nodes in the cluster as unschedulable", func() {
		nodes := ha.EventuallyGetNodes(clientset, t.Logs)
		var workerNodes []corev1.Node
		for _, node := range nodes.Items {
			if !ha.IsControlPlaneNode(node) {
				workerNodes = append(workerNodes, node)
			}
		}
		for i := 0; i < len(workerNodes)/2; i++ {
			workerNode := &workerNodes[i]
			workerNode.Spec.Unschedulable = true
			Eventually(func() bool {
				if _, err := clientset.CoreV1().Nodes().Update(context.TODO(), workerNode, metav1.UpdateOptions{}); err != nil {
					t.Logs.Errorf("Failed to mark node %s as unschedulable: %v", workerNode.Name, err)
					return false
				}
				t.Logs.Infof("Marked node %s as unschedulable", workerNode.Name)
				return true
			}, ha.WaitTimeout, ha.PollingInterval).Should(BeTrue())
		}
	})
})
