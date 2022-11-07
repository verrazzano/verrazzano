// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package validators

import (
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"strings"
	"time"
)

const (
	waitTimeout     = 3 * time.Minute
	pollingInterval = 10 * time.Second

	expectedJaegerErrorMessage = "the Jaeger Operator Helm chart value nameOverride cannot be overridden"
)

type jaegerIllegalUpdater struct{}

func (j jaegerIllegalUpdater) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1beta1.JaegerOperatorComponent{}
	}
	cr.Spec.Components.JaegerOperator.Enabled = &trueValue
	illegalOverride := `{"nameOverride": "testjaeger"}`
	illegalValuesObj := &apiextensionsv1.JSON{
		Raw: []byte(illegalOverride),
	}
	cr.Spec.Components.JaegerOperator.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.JaegerOperator.InstallOverrides.ValueOverrides,
		v1beta1.Overrides{Values: illegalValuesObj})
}

func (j jaegerIllegalUpdater) ModifyCR(cr *v1alpha1.Verrazzano) {
	// Attempt to make an illegal edit to the Jaeger configuration to ensure its component validation is working properly
	trueValue := true
	if cr.Spec.Components.JaegerOperator == nil {
		cr.Spec.Components.JaegerOperator = &v1alpha1.JaegerOperatorComponent{}
	}
	cr.Spec.Components.JaegerOperator.Enabled = &trueValue
	illegalOverride := `{"nameOverride": "testjaeger"}`
	illegalValuesObj := &apiextensionsv1.JSON{
		Raw: []byte(illegalOverride),
	}
	cr.Spec.Components.JaegerOperator.InstallOverrides.ValueOverrides = append(
		cr.Spec.Components.JaegerOperator.InstallOverrides.ValueOverrides,
		v1alpha1.Overrides{Values: illegalValuesObj})
}

var _ update.CRModifier = jaegerIllegalUpdater{}
var _ update.CRModifierV1beta1 = jaegerIllegalUpdater{}

// runValidatorTestV1Beta1 Attempt to use an illegal overrides value on the Jaeger operator configuration using the v1beta1 API
func runValidatorTestV1Beta1() {
	Eventually(func() bool {
		err := update.UpdateCRV1beta1(jaegerIllegalUpdater{})
		if err == nil {
			t.Logs.Info("Did not get an error on illegal update")
			return false
		}
		t.Logs.Infof("Update error: %s", err.Error())
		return strings.Contains(err.Error(), expectedJaegerErrorMessage)
	}, waitTimeout, pollingInterval).Should(BeTrue())
}

// runValidatorTestV1Alpha1 Attempt to use an illegal overrides value on the Jaeger operator configuration using the v1alpha1 API
func runValidatorTestV1Alpha1() {
	Eventually(func() bool {
		err := update.UpdateCR(jaegerIllegalUpdater{})
		if err == nil {
			t.Logs.Info("Did not get an error on illegal update")
			return false
		}
		t.Logs.Infof("Update error: %s", err.Error())
		return strings.Contains(err.Error(), expectedJaegerErrorMessage)
	}, waitTimeout, pollingInterval).Should(BeTrue())
}
