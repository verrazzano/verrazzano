// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/jaeger"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"github.com/verrazzano/verrazzano/tests/e2e/update/fluentd"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const (
	loggingNamespace = "verrazzano-logging"
	clusterName      = "opensearch"
	jaegerOSURLField = `
jaeger:
  spec:
    storage:
      options:
        es.server-urls:`
	longWaitTimeout      = 20 * time.Minute
	longPollingInterval  = 20 * time.Second
	shortPollingInterval = 10 * time.Second
	shortWaitTimeout     = 5 * time.Minute
)

var (
	t = framework.NewTestFramework("install")
)

type SwitchLoggingOutput struct {
	OpenSearchURL string
}

func (s *SwitchLoggingOutput) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.Fluentd = &v1beta1.FluentdComponent{
		OpenSearchURL: s.OpenSearchURL,
	}
	jaegerEnabled, _ := jaeger.IsJaegerEnabled()
	if jaegerEnabled {
		jaegerOSURLOverridesYaml := fmt.Sprintf(`%s %s`, jaegerOSURLField, s.OpenSearchURL)
		cr.Spec.Components.JaegerOperator.ValueOverrides = pkg.CreateOverridesOrDie(jaegerOSURLOverridesYaml)
	}
	t.Logs.Debugf("ModifiedV1beta1 CR: %s", marshalCRToString(cr.Spec))
}

var _ = t.AfterEach(func() {})

var _ = BeforeSuite(beforeSuite)

var beforeSuite = t.BeforeSuiteFunc(func() {

	t.Logs.Info(fmt.Sprintf("Creating %s namespace", loggingNamespace))
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			constants.LabelVerrazzanoNamespace: loggingNamespace,
		}
		return pkg.CreateNamespace(loggingNamespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Installing opensearch-operator and cluster")
	err := pkg.InstallOrUpdateOpenSearchOperator(t.Logs, 3, 3, 1)
	Expect(err).NotTo(HaveOccurred())
})

var _ = t.Describe("Verify opensearch and configure VZ", func() {
	t.It("verify opensearch pods are ready", func() {
		// Check all pods with opensearch prefix
		Eventually(func() bool {
			isReady, err := pkg.PodsRunning(loggingNamespace, []string{clusterName})
			if err != nil {
				return false
			}
			return isReady
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "OpenSearch failed to get to ready state")

		// Verify number of replicas for each nodepool
		pkg.EventuallyPodsReady(t.Logs, 3, 3, 1)
	})

	updateTime := time.Now()
	t.It("switch logging output", func() {
		// Update VZ CR to use new OS url for fluentd and jaeger, if enabled
		v1beta1Modifier := &SwitchLoggingOutput{OpenSearchURL: constants.DefaultOperatorOSURLWithNS}
		Eventually(func() bool {
			err := update.UpdateCRV1beta1(v1beta1Modifier)
			if err != nil {
				pkg.Log(pkg.Info, fmt.Sprintf("Update error: %v", err))
				return false
			}
			return true
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Failed to switch OS Url")
	})

	t.It("verify operator OS URL in fluentd", func() {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		Expect(err).NotTo(HaveOccurred())

		// Wait for VZ to be Ready after modifying the CR
		update.WaitForReadyState(kubeconfigPath, updateTime, longPollingInterval, longWaitTimeout)

		// Verify fluentd is up and ready with new OS URL
		Eventually(func() bool {
			return fluentd.ValidateDaemonsetV1beta1(constants.DefaultOperatorOSURLWithNS, constants.VerrazzanoESInternal, "")
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Fluentd not ready for %s", constants.DefaultOperatorOSURLWithNS)
	})
})

func marshalCRToString(cr interface{}) string {
	data, err := yaml.Marshal(cr)
	if err != nil {
		t.Logs.Errorf("Error marshalling CR to string")
		return ""
	}
	return string(data)
}
