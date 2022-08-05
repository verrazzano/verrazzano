// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaegeroperator

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout             = 3 * time.Minute
	pollingInterval         = 10 * time.Second
	jaegerOperatorName      = "jaeger-operator"
	minVZVersion            = "1.3.0"
	jaegerESIndexCleanerJob = "jaeger-operator-jaeger-es-index-cleaner"
)

var (
	jaegerOperatorCrds = []string{
		"jaegers.jaegertracing.io",
	}
	imagePrefix          = pkg.GetImagePrefix()
	operatorImage        = imagePrefix + "/verrazzano/" + jaegerOperatorName
	expectedJaegerImages = map[string]string{
		"JAEGER-AGENT-IMAGE":            imagePrefix + "/verrazzano/jaeger-agent",
		"JAEGER-COLLECTOR-IMAGE":        imagePrefix + "/verrazzano/jaeger-collector",
		"JAEGER-QUERY-IMAGE":            imagePrefix + "/verrazzano/jaeger-query",
		"JAEGER-INGESTER-IMAGE":         imagePrefix + "/verrazzano/jaeger-ingester",
		"JAEGER-ES-INDEX-CLEANER-IMAGE": imagePrefix + "/verrazzano/jaeger-es-index-cleaner",
		"JAEGER-ES-ROLLOVER-IMAGE":      imagePrefix + "/verrazzano/jaeger-es-rollover",
		"JAEGER-ALL-IN-ONE-IMAGE":       imagePrefix + "/verrazzano/jaeger-all-in-one",
	}
)

var t = framework.NewTestFramework("jaegeroperator")

func WhenJaegerOperatorEnabledIt(text string, args ...interface{}) {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(text, func() {
			Fail(err.Error())
		})
	}
	if pkg.IsJaegerOperatorEnabled(kubeconfig) {
		t.ItMinimumVersion(text, minVZVersion, kubeconfig, args...)
	}
	t.Logs.Infof("Skipping spec, Jaeger Operator is disabled")
}

var _ = t.Describe("Jaeger Operator", Label("f:platform-lcm.install"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the namespace exists
		// THEN we successfully find the namespace
		WhenJaegerOperatorEnabledIt("should have a verrazzano-monitoring namespace", func() {
			Eventually(func() (bool, error) {
				return pkg.DoesNamespaceExist(constants.VerrazzanoMonitoringNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the Jaeger pods are running
		// THEN we successfully find the running pods
		// For 1.3.0, only the jaeger-operator pod gets created and its status is validated
		// For 1.4.0 and later, jaeger-operator, jaeger-operator-jaeger-query, jaeger-operator-jaeger-collector
		//     pods gets created and their status is validated.
		WhenJaegerOperatorEnabledIt("should have running pods", func() {
			jaegerOperatorPodsRunning := func() bool {
				result, err := pkg.PodsRunning(constants.VerrazzanoMonitoringNamespace, []string{jaegerOperatorName})
				if err != nil {
					AbortSuite(fmt.Sprintf("Pod %v is not running in the namespace: %v, error: %v", jaegerOperatorName, constants.VerrazzanoMonitoringNamespace, err))
				}
				return result
			}
			Eventually(jaegerOperatorPodsRunning, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the default Jaeger images are from Verrazzano
		// THEN we see that the env is correctly populated
		WhenJaegerOperatorEnabledIt("should have the correct default Jaeger images", func() {
			verifyImages := func() bool {
				// Check if Jaeger operator is running with the expected Verrazzano Jaeger Operator image
				image, err := pkg.GetContainerImage(constants.VerrazzanoMonitoringNamespace, jaegerOperatorName, jaegerOperatorName)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Container %s is not running in the namespace: %s, error: %v", jaegerOperatorName, constants.VerrazzanoMonitoringNamespace, err))
					return false
				}
				if !strings.HasPrefix(image, operatorImage) {
					pkg.Log(pkg.Error, fmt.Sprintf("Container %s image %s is not running with the expected image %s in the namespace: %s", jaegerOperatorName, image, operatorImage, constants.VerrazzanoMonitoringNamespace))
					return false
				}
				// Check if Jaeger operator env has been set to use Verrazzano Jaeger images
				containerEnv, err := pkg.GetContainerEnv(constants.VerrazzanoMonitoringNamespace, jaegerOperatorName, jaegerOperatorName)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Not able to get the environment variables in the container %s, error: %v", jaegerOperatorName, err))
					return false
				}
				for name, val := range expectedJaegerImages {
					found := false
					for _, actualEnv := range containerEnv {
						if actualEnv.Name == name {
							if !strings.HasPrefix(actualEnv.Value, val) {
								pkg.Log(pkg.Error, fmt.Sprintf("The value %s of the env %s for the container %s does not have the image %s as expected",
									actualEnv.Value, actualEnv.Name, jaegerOperatorName, val))
								return false
							}
							found = true
						}
					}
					if !found {
						pkg.Log(pkg.Error, fmt.Sprintf("The env %s not set for the container %s", name, jaegerOperatorName))
						return false
					}
				}
				return true
			}
			Eventually(verifyImages, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check the CRDs created by Jaeger Operator
		// THEN we successfully find the Jaeger CRDs
		WhenJaegerOperatorEnabledIt("should have the correct Jaeger Operator CRDs", func() {
			verifyCRDList := func() (bool, error) {
				for _, crd := range jaegerOperatorCrds {
					exists, err := pkg.DoesCRDExist(crd)
					if err != nil || !exists {
						return exists, err
					}
				}
				return true, nil
			}
			Eventually(verifyCRDList, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator is installed
		// WHEN we check to make sure the Jaeger OpenSearch Index Cleaner cron job exists
		// THEN we successfully find the expected cron job
		WhenJaegerOperatorEnabledIt("should have a Jaeger OpenSearch Index Cleaner cron job", func() {
			Eventually(func() (bool, error) {
				create, err := pkg.IsJaegerInstanceCreated()
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error checking if Jaeger CR is available %s", err.Error()))
				}
				if create {
					return pkg.DoesCronJobExist(constants.VerrazzanoMonitoringNamespace, jaegerESIndexCleanerJob)
				}
				return false, nil
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

	})

})
