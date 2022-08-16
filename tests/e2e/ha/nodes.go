// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ha

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	WaitTimeout     = 10 * time.Minute
	PollingInterval = 10 * time.Second
)

func EventuallyGetNodes(cs *kubernetes.Clientset, log *zap.SugaredLogger) *corev1.NodeList {
	var nodes *corev1.NodeList
	var err error
	gomega.Eventually(func() bool {
		nodes, err = cs.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Errorf("failed to get nodes: %v", err)
			return false
		}
		return true
	}, WaitTimeout, PollingInterval).Should(gomega.BeTrue())
	return nodes
}

func EventuallySetNodeScheduling(cs *kubernetes.Clientset, name string, unschedulable bool, log *zap.SugaredLogger) {
	gomega.Eventually(func() bool {
		node, err := cs.CoreV1().Nodes().Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Failed to refresh node[%s]: %v", name, err)
			return false
		}
		node.Spec.Unschedulable = unschedulable
		if _, err := cs.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{}); err != nil {
			log.Errorf("Failed to update node[%s] scheduling: %v", name, err)
			return false
		}
		return true
	}).Should(gomega.BeTrue())
	log.Infof("Set node[%s].spec.unschedulable=%v", name, unschedulable)
}

func EventuallyEvictNode(cs *kubernetes.Clientset, name string, log *zap.SugaredLogger) {
	gomega.Eventually(func() bool {
		pods, err := cs.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", name),
		})
		if err != nil {
			log.Errorf("Failed to get pods for node[%s]: %v", name, err)
			return false
		}

		for i := range pods.Items {
			pod := &pods.Items[i]
			if err := cs.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{}); err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Failed to delete pod[%s] for node[%s]: %v", pod.Name, name, err)
					return false
				}
			}
		}
		return true
	}, WaitTimeout, PollingInterval).Should(gomega.BeTrue())
	log.Infof("Evicted node[%s]", name)
}

func IsControlPlaneNode(node corev1.Node) bool {
	_, ok := node.Labels["node-role.kubernetes.io/control-plane"]
	return ok
}
