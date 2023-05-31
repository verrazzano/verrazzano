// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/jaeger"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
	"time"
)

const (
	loggingNamespace = "verrazzano-logging"
	clusterName      = "opensearch"
	OSUrl            = "http://verrazzano-authproxy-opensearch-logging.verrazzano-system:8775"
	jaegerOSURLField = "jaeger.spec.storage.options.es.server-urls"
	timeout          = 15 * time.Minute
	pollInterval     = 10 * time.Second
)

var (
	t = framework.NewTestFramework("install")
)

type SwitchLoggingOutput struct {
	OpenSearchURL string
}

func (s *SwitchLoggingOutput) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.Fluentd.OpenSearchURL = s.OpenSearchURL
	jaegerEnabled, _ := jaeger.IsJaegerEnabled()
	if jaegerEnabled {
		jaegerOSURLOverridesYaml := fmt.Sprintf(`%s: %s`, jaegerOSURLField, s.OpenSearchURL)
		cr.Spec.Components.JaegerOperator.ValueOverrides = pkg.CreateOverridesOrDie(jaegerOSURLOverridesYaml)
	}
	t.Logs.Debugf("AuthProxyReplicasModifierV1beta1 CR: %s", marshalCRToString(cr.Spec))
}

var _ = t.AfterEach(func() {})

var _ = BeforeSuite(beforeSuite)

var beforeSuite = t.BeforeSuiteFunc(func() {

	t.Logs.Info(fmt.Sprintf("Creating %s namespace", loggingNamespace))
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
		}
		return pkg.CreateNamespace(loggingNamespace, nsLabels)
	}, timeout, pollInterval).ShouldNot(BeNil())

	t.Logs.Info("Install opensearch-operator and cluster")
	err := pkg.InstallOrUpdateOpenSearchOperator(t.Logs, 3, 3, 1)
	Expect(err).NotTo(HaveOccurred())
})

var _ = t.Describe("Verify opensearch and configure vz", func() {
	t.Describe("verify install", func() {
		t.It("opensearch pods are ready", func() {
			// Check all pods with opensearch prefix
			Eventually(func() bool {
				isReady, err := pkg.PodsRunning(loggingNamespace, []string{clusterName})
				if err != nil {
					return false
				}
				return isReady
			}, timeout, pollInterval).Should(BeTrue(), "OpenSearch failed to get to ready state")

			// Verify number of replicas for each nodepool
			pkg.EventuallyPodsReady(t.Logs, 3, 3, 1)
		})
	})

	t.Describe("configure vz cr", func() {
		t.It("switch logging output", func() {
			v1beta1Modifier := &SwitchLoggingOutput{OpenSearchURL: OSUrl}
			Eventually(func() bool {
				err := update.UpdateCRV1beta1(v1beta1Modifier)
				if err != nil {
					pkg.Log(pkg.Info, fmt.Sprintf("Update error: %v", err))
					return false
				}
				return true
			}, timeout, pollInterval).Should(BeTrue(), "Failed to switch OS Url")
		})
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
