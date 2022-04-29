package overrides

import (
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	v1 "k8s.io/api/core/v1"
)

var (
	t              = framework.NewTestFramework("overrides")
	enableData     = `{"prometheusOperator": {"enable": true}}`
	disableData    = `{"prometheusOperator": {"enable": false}}`
	deploymentName = "prometheus-operator-kube-p-operator"
)

type OverrideDisableModifier struct {
}

type OverrideEnableModifier struct {
}

var _ = t.Describe("Test helm overrides post install", func() {
	t.BeforeEach(func() {
		update.GetCR()
	})

	t.AfterEach(func() {

	})

	t.Context("disable prometheus operator", func() {
		// TODO: disable prometheus operator and check that deployment is not found
	})

	t.Context("re-enable prometheus operator", func() {
		// TODO: re-enable prometheus operator and check that re-install is successful
	})
})

func (o OverrideDisableModifier) ModifyCR(vz *vzapi.Verrazzano) {
	overrides := []vzapi.Overrides{
		{
			ConfigMapRef: &v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "po-values",
				},
				Key:      "values.yaml",
				Optional: nil,
			},
		},
	}
	WatchValues := true
	vz.Spec.Components.PrometheusOperator.ValueOverrides = overrides
	vz.Spec.Components.PrometheusOperator.MonitorChanges = &WatchValues
}

func (o OverrideEnableModifier) ModifyCR(vz *vzapi.Verrazzano) {
	overrides := vz.Spec.Components.PrometheusOperator.ValueOverrides
	overrides = append([]vzapi.Overrides{
		{
			ConfigMapRef: &v1.ConfigMapKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: "po-values",
				},
				Key:      "values1.yaml",
				Optional: nil,
			},
		},
	}, overrides...)
}
