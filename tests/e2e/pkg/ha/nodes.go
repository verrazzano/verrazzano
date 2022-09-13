// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ha

import (
	"context"
	"time"

	"github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	WaitTimeout         = 10 * time.Minute
	longWaitTimeout     = 20 * time.Minute
	PollingInterval     = 10 * time.Second
	longPollingInterval = 30 * time.Second
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

func IsControlPlaneNode(node corev1.Node) bool {
	_, ok := node.Labels["node-role.kubernetes.io/control-plane"]
	return ok
}
