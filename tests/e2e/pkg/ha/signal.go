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

const (
	shutdownSignalName      = "ha-shutdown-signal"
	shutdownSignalNamespace = "default"
)

func EventuallyCreateShutdownSignal(cs *kubernetes.Clientset, log *zap.SugaredLogger) {
	gomega.Eventually(func() bool {
		shutdownSignal := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: shutdownSignalNamespace,
				Name:      shutdownSignalName,
			},
		}
		if _, err := cs.CoreV1().Secrets(shutdownSignalNamespace).Create(context.TODO(), shutdownSignal, metav1.CreateOptions{}); err != nil {
			log.Errorf("Failed to create shutdown signal: %v", err)
			return false
		}
		return true
	}).Should(gomega.BeTrue())
	log.Infof("Created shutdown signal %s", shutdownSignalName)
}

func IsShutdownSignalSet(cs *kubernetes.Clientset) bool {
	_, err := cs.CoreV1().Secrets(shutdownSignalNamespace).Get(context.TODO(), shutdownSignalName, metav1.GetOptions{})
	return err == nil
}
