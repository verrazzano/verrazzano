// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package jaeger

import (
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

const (
	waitTimeout               = 20 * time.Minute
	pollingInterval           = 10 * time.Second
	jaegerComponentLabel      = "app.kubernetes.io/name"
	jaegerOperatorLabelValue  = "jaeger-operator"
	jaegerCollectorLabelValue = "jaeger-operator-jaeger-collector"
	jaegerQueryLabelValue     = "jaeger-operator-jaeger-query"
)

var trueValue = true

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
