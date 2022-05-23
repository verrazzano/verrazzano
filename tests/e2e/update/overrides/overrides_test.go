// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package overrides

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	corev1 "k8s.io/api/core/v1"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitTimeout                          = 5 * time.Minute
	pollingInterval                      = 5 * time.Second
	overrideConfigMapSecretName   string = "test-overrides-1"
	verrazzanoMonitoringNamespace string = "verrazzano-monitoring"
	dataKey                       string = "values.yaml"
	overrideKey                   string = "override"
	overrideOldValue              string = "true"
	overrideNewValue              string = "false"
	deploymentName                string = "prometheus-operator-kube-p-operator"
)

var (
	t = framework.NewTestFramework("overrides")
)

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || ginkgo.CurrentSpecReport().Failed()
})

type PrometheusOperatorOverridesModifier struct {
}

type PrometheusOperatorDefaultModifier struct {
}

func (d PrometheusOperatorDefaultModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.PrometheusOperator != nil {
		if cr.Spec.Components.PrometheusOperator.ValueOverrides != nil {
			cr.Spec.Components.PrometheusOperator.ValueOverrides = nil
		}
	}
}

func (o PrometheusOperatorOverridesModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.PrometheusOperator == nil {
		cr.Spec.Components.PrometheusOperator = &vzapi.PrometheusOperatorComponent{}
	}
	var trueVal = true
	overrides := []vzapi.Overrides{
		{
			ConfigMapRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: overrideConfigMapSecretName,
				},
				Key:      dataKey,
				Optional: &trueVal,
			},
		},
		{
			SecretRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: overrideConfigMapSecretName,
				},
				Key:      dataKey,
				Optional: &trueVal,
			},
		},
	}
	cr.Spec.Components.PrometheusOperator.Enabled = &trueVal
	cr.Spec.Components.PrometheusOperator.MonitorChanges = &trueVal
	cr.Spec.Components.PrometheusOperator.ValueOverrides = overrides
}

var _ = t.BeforeSuite(func() {
	m := PrometheusOperatorOverridesModifier{}
	update.UpdateCR(m)
	_ = update.GetCR()
})

var _ = t.AfterSuite(func() {
	m := PrometheusOperatorDefaultModifier{}
	update.UpdateCR(m)
	_ = update.GetCR()
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

var _ = t.Describe("Post Install Overrides", func() {

	t.Context("Test overrides creation", func() {
		// Create the overrides resources listed in Verrazzano and verify
		// that the values have been applied to promtheus-operator
		t.Context("Create Overrides", func() {
			t.It("Create ConfigMap", func() {
				testConfigMap.Data[dataKey] = oldCMData
				gomega.Eventually(func() error {
					return pkg.CreateConfigMap(&testConfigMap)
				}, waitTimeout, pollingInterval).Should(gomega.BeNil())
			})

			t.It("Create Secret", func() {
				testSecret.StringData[dataKey] = oldSecretData
				gomega.Eventually(func() error {
					return pkg.CreateSecret(&testSecret)
				}, waitTimeout, pollingInterval).Should(gomega.BeNil())
			})
		})

		t.It("Verify override values are applied", func() {
			gomega.Eventually(func() bool {
				return checkValues(overrideOldValue)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		// Verify that re-install succeeds
		t.It("Verify Verrazzano re-install is successful", func() {
			gomega.Eventually(func() error {
				return vzReady()
			}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")
		})
	})

	t.Context("Test overrides update", func() {
		// Update the overrides resources listed in Verrazzano and verify
		// that the new values have been applied to promtheus-operator
		t.Context("Update Overrides", func() {
			t.It("Update ConfigMap", func() {
				testConfigMap.Data[dataKey] = newCMData
				gomega.Eventually(func() error {
					return pkg.UpdateConfigMap(&testConfigMap)
				}, waitTimeout, pollingInterval).Should(gomega.BeNil())
			})

			t.It("Update Secret", func() {
				testSecret.StringData[dataKey] = newSecretData
				gomega.Eventually(func() error {
					return pkg.UpdateSecret(&testSecret)
				}, waitTimeout, pollingInterval).Should(gomega.BeNil())
			})
		})

		t.It("Verify override values are applied", func() {
			gomega.Eventually(func() bool {
				return checkValues(overrideNewValue)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		// Verify that re-install succeeds
		t.It("Verify Verrazzano re-install is successful", func() {
			gomega.Eventually(func() error {
				return vzReady()
			}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")
		})
	})

	t.Context("Test overrides deletion", func() {
		// Delete the resources and verify that the deleted
		// values are now unapplied
		t.It("Delete Resources", func() {
			deleteOverrides()
		})

		t.It("Verify deleted values are removed", func() {
			gomega.Eventually(func() bool {
				pods, err := pkg.GetPodsFromSelector(nil, constants.VerrazzanoMonitoringNamespace)
				if err != nil {
					return false
				}
				for _, pod := range pods {
					if strings.Contains(pod.Name, deploymentName) {
						_, foundLabel := pod.Labels[overrideKey]
						_, foundAnnotation := pod.Annotations[overrideKey]
						if !foundLabel && !foundAnnotation {
							return true
						}
					}
				}
				return false
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		// Verify that re-install succeeds
		t.It("Verify Verrazzano re-install is successful", func() {
			gomega.Eventually(func() error {
				return vzReady()
			}, waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")
		})
	})
})

func deleteOverrides() {
	err0 := pkg.DeleteConfigMap(constants.DefaultNamespace, overrideConfigMapSecretName)
	if err0 != nil && !k8serrors.IsNotFound(err0) {
		ginkgo.AbortSuite("Failed to delete ConfigMap")
	}

	err1 := pkg.DeleteSecret(constants.DefaultNamespace, overrideConfigMapSecretName)
	if err1 != nil && !k8serrors.IsNotFound(err1) {
		ginkgo.AbortSuite("Failed to delete Secret")
	}

}

func vzReady() error {
	cr, err := pkg.GetVerrazzano()
	if err != nil {
		return err
	}
	if cr.Status.State != vzapi.VzStateReady {
		return fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
	}
	return nil
}

func checkValues(overrideValue string) bool {
	labelMatch := map[string]string{overrideKey: overrideValue}
	pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{
		MatchLabels: labelMatch,
	}, verrazzanoMonitoringNamespace)
	if err != nil {
		ginkgo.AbortSuite(fmt.Sprintf("Label override not found for the Prometheus Operator pod in namespace %s: %v", verrazzanoMonitoringNamespace, err))
	}
	foundAnnotation := false
	for _, pod := range pods {
		if val, ok := pod.Annotations[overrideKey]; ok && val == overrideValue {
			foundAnnotation = true
		}
	}
	return len(pods) == 1 && foundAnnotation
}
