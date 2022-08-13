// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	"time"
)

const (
	waitTimeout               = 20 * time.Minute
	pollingInterval           = 10 * time.Second
	jaegerComponentLabel      = "app.kubernetes.io/name"
	jaegerOperatorLabelValue  = "jaeger-operator"
	jaegerCollectorLabelValue = "jaeger-operator-jaeger-collector"
	jaegerQueryLabelValue     = "jaeger-operator-jaeger-query"
)

var (
	// Initialize the Test Framework
	t         = framework.NewTestFramework("update Jaeger operator")
	trueValue = true
	start     = time.Now()
)

type JaegerOperatorCleanupModifier struct {
}
type JaegerOperatorEnabledModifier struct {
}

func (u JaegerOperatorCleanupModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.JaegerOperator = &vzapi.JaegerOperatorComponent{}
}

func (u JaegerOperatorEnabledModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.JaegerOperator = &vzapi.JaegerOperatorComponent{
		Enabled: &trueValue,
	}
}

func WhenJaegerOperatorEnabledIt(text string, args ...interface{}) {
	kubeconfig, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		t.It(text, func() {
			ginkgo.Fail(err.Error())
		})
	}
	if pkg.IsJaegerOperatorEnabled(kubeconfig) {
		t.ItMinimumVersion(text, "1.3.0", kubeconfig, args...)
	}
	t.Logs.Infof("Skipping spec, Jaeger Operator is disabled")
}

var _ = t.BeforeSuite(func() {
	m := JaegerOperatorEnabledModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
})
