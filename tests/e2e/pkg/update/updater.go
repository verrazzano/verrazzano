// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

type CRModifier interface {
	ModifyCR(cr *vzapi.Verrazzano)
}

type CRModifierV1beta1 interface {
	ModifyCRV1beta1(cr *v1beta1.Verrazzano)
}

// GetCR gets the CR.  If it is not "Ready", wait for up to 5 minutes for it to be "Ready".
func GetCR() *vzapi.Verrazzano {
	var vz *vzapi.Verrazzano
	// Wait for the CR to be Ready
	gomega.Eventually(func() error {
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			return err
		}
		if cr.Status.State != vzapi.VzStateReady {
			return fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
		}
		vz = cr
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")
	return vz
}

// GetCRV1beta1 gets the CR.  If it is not "Ready", wait for up to 5 minutes for it to be "Ready".
func GetCRV1beta1() *v1beta1.Verrazzano {
	// Wait for the CR to be Ready
	var vz *v1beta1.Verrazzano
	gomega.Eventually(func() error {
		cr, err := pkg.GetVerrazzanoV1beta1()
		if err != nil {
			return err
		}
		if cr.Status.State != v1beta1.VzStateReady {
			return fmt.Errorf("v1beta1 CR in state %s, not Ready yet", cr.Status.State)
		}
		vz = cr
		return nil
	}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano v1beta1 CR with Ready state")

	return vz
}

// UpdateCR updates the CR with the given CRModifier
// - if the CRModifier implements rest.WarningHandler it will be added to the client config
//
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
	path, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	config, err := k8sutil.GetKubeConfigGivenPath(path)
	if err != nil {
		return err
	}
	addWarningHandlerIfNecessary(m, config)
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		return err
	}
	vzClient := client.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}

// UpdateCRV1beta1WithRetries updates the CR with the given CRModifierV1beta1.
// First it waits for CR to be "Ready" before using the specified CRModifierV1beta1 modifies the CR.
// Then, it updates the modified.
// Any error during the process will cause Ginkgo Fail.
func UpdateCRV1beta1WithRetries(m CRModifierV1beta1, pollingInterval, waitTime time.Duration) {
	// Update the CR
	gomega.Eventually(func() bool {
		// GetCRV1beta1 gets the CR using v1beta1 client.
		cr := GetCRV1beta1()

		// Modify the CR
		m.ModifyCRV1beta1(cr)

		path, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		config, err := k8sutil.GetKubeConfigGivenPath(path)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		addWarningHandlerIfNecessary(m, config)
		client, err := vpoClient.NewForConfig(config)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
		_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		return true
	}).WithPolling(pollingInterval).WithTimeout(waitTime).Should(gomega.BeTrue())
}

func addWarningHandlerIfNecessary(m interface{}, config *rest.Config) {
	warningHandler, implementsWarningHandler := m.(rest.WarningHandler)
	if implementsWarningHandler {
		config.WarningHandler = warningHandler
	}
}

// UpdateCRV1beta1 updates the CR with the given CRModifierV1beta1.
// - if the modifier implements rest.WarningHandler it will be added to the client config
//
// First it waits for CR to be "Ready" before using the specified CRModifierV1beta1 modifies the CR.
// Then, it updates the modified.
// Any error during the process will cause Ginkgo Fail.
func UpdateCRV1beta1(m CRModifierV1beta1) error {
	// GetCRV1beta1 gets the CR using v1beta1 client.
	cr := GetCRV1beta1()

	// Modify the CR
	m.ModifyCRV1beta1(cr)

	// Update the CR
	var err error
	path, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return err
	}
	config, err := k8sutil.GetKubeConfigGivenPath(path)
	if err != nil {
		return err
	}
	addWarningHandlerIfNecessary(m, config)
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		return err
	}
	vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	return err
}

// UpdateCRWithRetries updates the CR with the given CRModifier.
// - if the modifier implements rest.WarningHandler it will be added to the client config
//
// If the update fails, it retries by getting the latest version of CR and applying the same
// update till it succeeds or timesout.
func UpdateCRWithRetries(m CRModifierV1beta1, pollingInterval, timeout time.Duration) {
	RetryUpdate(m, "", true, pollingInterval, timeout)
}

// UpdateCRWithRetriesV1Alpha1 updates the CR with the given CRModifier.
// - if the modifier implements rest.WarningHandler it will be added to the client config
//
// If the update fails, it retries by getting the latest version of CR and applying the same
// update till it succeeds or timesout.
func UpdateCRWithRetriesV1Alpha1(m CRModifier, pollingInterval, timeout time.Duration) {
	RetryUpdateV1Alpha1(m, "", true, pollingInterval, timeout)
}

// UpdateCRWithPlugins updates the CR with the given CRModifier.
// update till it succeeds or timesout.
func UpdateCRWithPlugins(m CRModifierV1beta1, pollingInterval, timeout time.Duration) {
	UpdatePlugins(m, "", true, pollingInterval, timeout)
}

// UpdatePlugins tries update with kubeconfigPath
func UpdatePlugins(m CRModifierV1beta1, kubeconfigPath string, waitForReady bool, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		var err error
		if kubeconfigPath == "" {
			kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
		}

		cr, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		// Modify the CR
		m.ModifyCRV1beta1(cr)
		config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		addWarningHandlerIfNecessary(m, config)
		client, err := vpoClient.NewForConfig(config)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
		_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		return true
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

// RetryUpdate tries update with kubeconfigPath
// - if the modifier implements rest.WarningHandler it will be added to the client config
func RetryUpdate(m CRModifierV1beta1, kubeconfigPath string, waitForReady bool, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		var err error
		if kubeconfigPath == "" {
			kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
		}
		if waitForReady {
			WaitForReadyState(kubeconfigPath, time.Time{}, pollingInterval, timeout)
		}
		cr, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		// Modify the CR
		m.ModifyCRV1beta1(cr)
		config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		addWarningHandlerIfNecessary(m, config)
		client, err := vpoClient.NewForConfig(config)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
		_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		if waitForReady {
			// Wait till the resource edit is complete and the verrazzano custom resource comes to ready state
			WaitForReadyState(kubeconfigPath, time.Now(), pollingInterval, timeout)
		}
		return true
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

// RetryUpdateV1alpha1 tries update with kubeconfigPath
// - if the modifier implements rest.WarningHandler it will be added to the client config
func RetryUpdateV1Alpha1(m CRModifier, kubeconfigPath string, waitForReady bool, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		var err error
		if kubeconfigPath == "" {
			kubeconfigPath, err = k8sutil.GetKubeConfigLocation()
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
		}
		if waitForReady {
			WaitForReadyState(kubeconfigPath, time.Time{}, pollingInterval, timeout)
		}
		cr, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		// Modify the CR
		m.ModifyCR(cr)
		config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		addWarningHandlerIfNecessary(m, config)
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
		if waitForReady {
			// Wait till the resource edit is complete and the verrazzano custom resource comes to ready state
			WaitForReadyState(kubeconfigPath, time.Now(), pollingInterval, timeout)
		}
		return true
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

func UpdateCRExpectError(m CRModifierV1beta1) error {
	cr, err := pkg.GetVerrazzanoV1beta1()
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return err
	}
	// Modify the CR
	m.ModifyCRV1beta1(cr)

	// Update the CR
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return err
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return err
	}
	addWarningHandlerIfNecessary(m, config)
	client, err := vpoClient.NewForConfig(config)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return err
	}
	vzClient := client.VerrazzanoV1beta1().Verrazzanos(cr.Namespace)
	_, err = vzClient.Update(context.TODO(), cr, metav1.UpdateOptions{})
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return err
	}
	return nil
}

// IsCRReady return true if the verrazzano custom resource is in ready state after an update operation false otherwise
func IsCRReadyAfterUpdate(cr *v1beta1.Verrazzano, updatedTime time.Time) bool {
	if cr == nil || cr.Status.State != v1beta1.VzStateReady {
		pkg.Log(pkg.Error, "VZ CR is nil or not in ready state")
		return false
	}
	for _, condition := range cr.Status.Conditions {
		pkg.Log(pkg.Info, fmt.Sprintf("Checking if condition of type '%s', transitioned at '%s' is for the expected update",
			condition.Type, condition.LastTransitionTime))
		if (condition.Type == v1beta1.CondInstallComplete || condition.Type == v1beta1.CondUpgradeComplete) && condition.Status == corev1.ConditionTrue {
			// check if the transition time is post the time of update
			transitionTime, err := time.Parse(time.RFC3339, condition.LastTransitionTime)
			if err != nil {
				pkg.Log(pkg.Error, "Unable to parse the transition time '"+condition.LastTransitionTime+"'")
				return false
			}
			if transitionTime.After(updatedTime) {
				pkg.Log(pkg.Info, "Update operation completed at "+transitionTime.String())
				return true
			}
		}
		pkg.Log(pkg.Error, fmt.Sprintf("Could not find condition of type '%s' or '%s', transitioned after '%s'",
			vzapi.CondInstallComplete, vzapi.CondUpgradeComplete, updatedTime.String()))
	}
	// Return true if the state is ready and there are no conditions updated in the status.
	return len(cr.Status.Conditions) == 0
}

// WaitForReadyState waits till the verrazzano custom resource becomes ready or times out
func WaitForReadyState(kubeconfigPath string, updateTime time.Time, pollingInterval, timeout time.Duration) {
	gomega.Eventually(func() bool {
		cr, err := pkg.GetVerrazzanoInstallResourceInClusterV1beta1(kubeconfigPath)
		if err != nil {
			pkg.Log(pkg.Error, err.Error())
			return false
		}
		return IsCRReadyAfterUpdate(cr, updateTime)
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}
