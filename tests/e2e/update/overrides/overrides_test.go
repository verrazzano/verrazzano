// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package overrides

import (
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os/exec"
	"time"
)

const (
	waitTimeout                          = 5 * time.Minute
	pollingInterval                      = 5 * time.Second
	overrideConfigMapSecretName   string = "test-overrides"
	verrazzanoMonitoringNamespace string = "verrazzano-monitoring"
	overrideKey                   string = "override"
	overrideValue                 string = "false"
	deploymentName                string = "prometheus-operator-kube-p-operator"
)

var (
	labelMatch = map[string]string{overrideKey: overrideValue}
	t          = framework.NewTestFramework("overrides")
)

var _ = t.Describe("Post Install Overrides Test", func() {
	t.Context("Update install override values", func() {
		t.It("Update overrides ConfigMap", func() {
			updateOverrides()
		})

		t.It("Check Verrazzano reaches ready state", func() {
			gomega.Eventually(vzReady(), waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")
		})

		t.It("Check new pod label and annotation", func() {
			gomega.Eventually(func() bool {
				_, err := pkg.GetConfigMap(overrideConfigMapSecretName, constants.DefaultNamespace)
				if err == nil {
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
				} else if !k8serrors.IsNotFound(err) {
					ginkgo.AbortSuite(fmt.Sprintf("Error retrieving the override ConfigMap: %v", err))
				}
				return true
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})
	t.Context("Delete Overrides", func() {
		t.It("Delete Resources", func() {
			deleteOverrides()
		})

		t.It("Check Verrazzano reaches ready state", func() {
			gomega.Eventually(vzReady(), waitTimeout, pollingInterval).Should(gomega.BeNil(), "Expected to get Verrazzano CR with Ready state")
		})

		t.It("Check deleted label and annotation have been removed", func() {
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
	})
})

func updateOverrides() {
	output, err := exec.Command("/bin/sh", "update_overrides.sh").Output()
	if err != nil {
		log.Fatalf("Error in updating ConfigMap")
	}
	log.Printf(string(output))
}

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
	if cr != nil && cr.Status.State != vzapi.VzStateReady {
		return fmt.Errorf("CR in state %s, not Ready yet", cr.Status.State)
	}
	return nil
}
