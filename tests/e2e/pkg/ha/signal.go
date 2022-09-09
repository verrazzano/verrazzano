// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ha

import (
	"context"
	"time"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	shutdownSignalName      = "ha-shutdown-signal"
	shutdownSignalNamespace = "default"

	shortWaitTimeout     = time.Minute
	shortPollingInterval = 5 * time.Second
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
	}).WithTimeout(shortWaitTimeout).WithPolling(shortPollingInterval).Should(gomega.BeTrue())
	log.Infof("Created shutdown signal %s", shutdownSignalName)
}

func IsShutdownSignalSet(cs *kubernetes.Clientset) bool {
	_, err := cs.CoreV1().Secrets(shutdownSignalNamespace).Get(context.TODO(), shutdownSignalName, metav1.GetOptions{})
	return err == nil
}

// RunningUntilShutdownIt runs the test function repeatedly until the shutdown signal is set. If the "runContinues" flag is false, the
// test is only run one time.
func RunningUntilShutdownIt(t *framework.TestFramework, description string, clientset *kubernetes.Clientset, runContinuous bool, test func()) {
	t.It(description, func() {
		for {
			test()
			// break out of the loop if we are not running the suite continuously,
			// or the shutdown signal is set
			if !runContinuous || IsShutdownSignalSet(clientset) {
				t.Logs.Info("Shutting down...")
				break
			}
		}
	})
}
