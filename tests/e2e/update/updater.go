// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"time"

	ginkgov2 "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

type CRModifier interface {
	ModifyCR(cr *vzapi.Verrazzano)
}

// GetCR gets the CR.  If it is not "Ready", wait for up to 5 minutes for it to be "Ready".
func GetCR() *vzapi.Verrazzano {
	// Wait for the CR to be Ready
	gomega.Eventually(func() error {
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			return err
		}
		if cr.Status.State != vzapi.VzStateReady {
			return fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
		}
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")

	// Get the CR
	cr, err := pkg.GetVerrazzano()
	if err != nil {
		ginkgov2.Fail(err.Error())
	}
	if cr == nil {
		ginkgov2.Fail("CR is nil")
	}

	return cr
}

// UpdateCR updates the CR with the given CRModifier.
// First it waits for CR to be "Ready" before using the specified CRModifier modifies the CR.
// Then, it updates the modified.
// Any error during the process will cause Ginkgo Fail.
func UpdateCR(m CRModifier) error {
	// Get the CR
	cr := GetCR()

	// Modify the CR
	m.ModifyCR(cr)

	// Update the CR
	var err error
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		ginkgov2.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		ginkgov2.Fail(err.Error())
	}
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}

// UpdateCRWithRetries updates the CR with the given CRModifier.
// If the update fails, it retries by getting the latest version of CR and applying the same
// update till it succeeds or timesout.
func UpdateCRWithRetries(m CRModifier, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		// Wait till the custom resource becomes ready.
		WaitForReadyState(time.Time{}, pollingInterval, timeout)
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		// Modify the CR
		m.ModifyCR(cr)

		// Update the CR
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		client, err := vpoClient.NewForConfig(config)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		vzClient := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace)
		_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		// Wait till the resource edit is complete and the verrazzano custom resource comes to ready state
		WaitForReadyState(time.Now(), pollingInterval, timeout)
		return true
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

// IsCRReady return true if the verrazzano custom resource is in ready state after an update operation false otherwise
func IsCRReadyAfterUpdate(cr *vzapi.Verrazzano, updatedTime time.Time) bool {
	if cr == nil || cr.Status.State != vzapi.VzStateReady {
		pkg.Log(pkg.Error, "VZ CR is nil or not in ready state")
		return false
	}
	for _, condition := range cr.Status.Conditions {
		if condition.Type == vzapi.CondInstallComplete && condition.Status == corev1.ConditionTrue {
			// check if the transistion time is post the time of update
			transitionTime, err := time.Parse(time.RFC3339, condition.LastTransitionTime)
			if err != nil {
				pkg.Log(pkg.Error, "Unable to parse the transistion time '"+condition.LastTransitionTime+"'")
				return false
			}
			if transitionTime.After(updatedTime) {
				pkg.Log(pkg.Info, "Update operation completed at "+updatedTime.String())
				return true
			}
		}
	}
	// Return true if the state is ready and there are no conditions updated in the status.
	return len(cr.Status.Conditions) == 0
}

// WaitForReadyState waits till the verrazzano custom resource becomes ready or times out
func WaitForReadyState(updateTime time.Time, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		return IsCRReadyAfterUpdate(cr, updateTime)
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}
