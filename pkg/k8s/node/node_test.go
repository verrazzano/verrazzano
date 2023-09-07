// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package node

import (
	"context"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// TestGetK8sNodeList tests getting a list of Kubernetes nodes
func TestGetK8sNodeList(t *testing.T) {
	asserts := assert.New(t)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&v1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1",
			},
		}, &v1.Node{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Node",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "node2",
			},
		}).Build()
	nodeList, err := GetK8sNodeList(client)
	asserts.NoError(err)
	asserts.Equal(2, len(nodeList.Items))
	asserts.Equal("node1", nodeList.Items[0].Name)
	asserts.Equal("node2", nodeList.Items[1].Name)
}

func TestSetSingleNodeTaints(t *testing.T) {
	ctx := context.TODO()
	workerNode := createTestNode("worker")
	masterNode := createTestNode("master", v1.Taint{
		Key:    controlPlaneTaint,
		Effect: "NoSchedule",
	}, v1.Taint{
		Key:    masterTaint,
		Effect: "NoSchedule",
	}, v1.Taint{
		Key:    "abc",
		Effect: "xyz",
	})
	cli := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(workerNode, masterNode).Build()

	err := SetControlPlaneScheduling(ctx, cli)
	assert.NoError(t, err)

	nodes := &v1.NodeList{}
	err = cli.List(ctx, nodes)
	assert.NoError(t, err)
	assert.Len(t, nodes.Items, 2)
	for _, node := range nodes.Items {
		for _, taint := range node.Spec.Taints {
			assert.NotEqual(t, controlPlaneTaint, taint.Key)
			assert.NotEqual(t, masterTaint, taint.Key)
		}
	}
}

func createTestNode(name string, taints ...v1.Taint) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.NodeSpec{
			Taints: taints,
		},
	}
}
