// Copyright (C) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package infra

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"sigs.k8s.io/yaml"
	"time"

	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var trueValue = true
var falseValue = false

type FluentBitOperatorEnabledModifier struct {
}

func (u FluentBitOperatorEnabledModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.FluentOperator = &vzapi.FluentOperatorComponent{
		Enabled: &trueValue,
	}
	cr.Spec.Components.FluentbitOpensearchOutput = &vzapi.FluentbitOpensearchOutputComponent{
		Enabled: &trueValue,
	}
	cr.Spec.Components.Fluentd = &vzapi.FluentdComponent{
		Enabled: &falseValue,
	}
	t.Logs.Debugf("FluentBitOperatorEnabledModifier CR: %v", marshalCRToString(cr.Spec))
}

const (
	longWaitTimeout             = 20 * time.Minute
	longPollingInterval         = 20 * time.Second
	shortPollingInterval        = 10 * time.Second
	shortWaitTimeout            = 5 * time.Minute
	fluentBitComponentLabel     = "app.kubernetes.io/name"
	fluentBitOperatorLabelValue = "fluent-operator"
	fluentBitLabelValue         = "fluent-bit"
)

var (
	t = framework.NewTestFramework("infra")
)

var _ = t.AfterEach(func() {})

var _ = BeforeSuite(beforeSuite)

var beforeSuite = t.BeforeSuiteFunc(func() {

	nodeCount, err := pkg.GetNodeCount()
	if err != nil {
		Fail(err.Error())
	}

	m := FluentBitOperatorEnabledModifier{}
	update.UpdateCRWithRetries(m, longPollingInterval, longWaitTimeout)

	// GIVEN a VZ custom resource in dev profile,
	// WHEN FluentBit operator is enabled,
	// THEN pods for fluentbit-operator components gets created.
	update.ValidatePods(fluentBitOperatorLabelValue, fluentBitComponentLabel, constants.VerrazzanoSystemNamespace, 1, false)
	update.ValidatePods(fluentBitLabelValue, fluentBitComponentLabel, constants.VerrazzanoSystemNamespace, nodeCount, false)
})

// GIVEN a VZ custom resource in dev profile,
// WHEN FluentBit operator is enabled, and Fluentd is disabled
// THEN expect the Opensearch index for the verrazzano-system exists
var _ = t.Describe("Verify FluentBit Post Install infra", func() {
	t.It("verrazzano-system index is present", func() {
		Eventually(func() bool {
			return pkg.LogIndexFound("verrazzano-system")
		}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())
	})

	// GIVEN FluentBit operator is enabled, and Fluentd is disabled
	// WHEN the log records are retrieved from the Opensearch verrazzano-system index
	// THEN verify that at least one recent log record is found
	t.It("Verify recent Opensearch log record exists", func() {
		Eventually(func() bool {
			return pkg.LogRecordFound("verrazzano-system", time.Now().Add(-5*time.Minute), map[string]string{
				"kubernetes.namespace_name": "verrazzano-system"})
		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record for k8s-sidecar container")
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
