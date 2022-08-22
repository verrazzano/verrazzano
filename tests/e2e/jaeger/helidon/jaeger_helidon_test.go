// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/jaeger"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

const (
	testAppComponentFilePath     = "testdata/jaeger/helidon/helidon-tracing-comp.yaml"
	testAppConfigurationFilePath = "testdata/jaeger/helidon/helidon-tracing-app.yaml"
)

var (
	t                        = framework.NewTestFramework("jaeger-helidon")
	generatedNamespace       = pkg.GenerateNamespace("jaeger-tracing")
	expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
	beforeSuitePassed        = false
	failed                   = false
	start                    = time.Now()
	helloHelidonServiceName  = "hello-helidon"
)

var _ = t.BeforeSuite(func() {
	start = time.Now()
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		AbortSuite("Unable to get the default kubeconfig path")
	}
	jaeger.DeployApplication(namespace, testAppComponentFilePath, testAppConfigurationFilePath, expectedPodsHelloHelidon)

	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	err = pkg.GenerateTrafficForTraces(namespace, "", "greet", kubeconfigPath)
	if err != nil {
		AbortSuite("Unable to send traffic requests to generate traces")
	}
	beforeSuitePassed = true
})

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	// undeploy the application here
	start := time.Now()

	jaeger.UndeployApplication(namespace, testAppComponentFilePath, testAppConfigurationFilePath, expectedPodsHelloHelidon)
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Helidon App with Jaeger Traces", Label("f:jaeger.helidon-workload"), func() {
	t.Context("after successful installation", func() {
		// GIVEN the Jaeger Operator is enabled and a sample application is installed,
		// WHEN we check for traces for that service,
		// THEN we are able to get the traces
		jaeger.WhenJaegerOperatorEnabledIt(t, "traces for the helidon app should be available when queried from Jaeger", func() {
			kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				Fail(err.Error())
			}
			validatorFn := pkg.ValidateApplicationTracesInCluster(kubeconfigPath, start, helloHelidonServiceName, "local")
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})

		// GIVEN the Jaeger Operator component is enabled,
		// WHEN a sample application is installed,
		// THEN the traces are found in OpenSearch Backend
		jaeger.WhenJaegerOperatorEnabledIt(t, "traces for the helidon app should be available in the OS backend storage.", func() {
			validatorFn := pkg.ValidateApplicationTracesInOS(start, helloHelidonServiceName)
			Eventually(validatorFn).WithPolling(shortPollingInterval).WithTimeout(shortWaitTimeout).Should(BeTrue())
		})
	})

})
