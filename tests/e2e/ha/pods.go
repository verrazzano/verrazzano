// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ha

import (
	"context"

	"github.com/onsi/gomega"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func EventuallyPodsReady(log *zap.SugaredLogger, cs *kubernetes.Clientset) {
	var pods *corev1.PodList
	gomega.Eventually(func() bool {
		var err error
		pods, err = cs.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Info("Failed to get pods: %v", err)
			return false
		}

		// Assume all pods are ready.  If debug enabled, log status of each pod that is not ready yet
		returnValue := true
		for _, pod := range pods.Items {
			if !IsPodReadyOrCompleted(pod) {
				log.Debugf("Pod [%s] in namespace [%s] not ready or completed [%s]", pod.Name, pod.Namespace, string(pod.Status.Phase))
				returnValue = false
			}
		}
		return returnValue

	}, WaitTimeout, PollingInterval).Should(gomega.BeTrue())
}

func IsPodReadyOrCompleted(pod corev1.Pod) bool {
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
