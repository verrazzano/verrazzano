package overrides

import (
	"context"
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoClient "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

const (
	waitTimeOut     = 5 * time.Minute
	pollingInterval = 5 * time.Second
)

var (
	t             = framework.NewTestFramework("overrides")
	overrideData0 = `{"prometheusOperator":{"podLabels":{"override": "true"}}}`
	overrideData1 = `{"prometheusOperator":{"podLabels":{"override": "false"}}}`
)

type OverrideModifier struct{}

var _ = t.BeforeSuite(func() {
	t.Logs.Info("Create ConfigMap")
	gomega.Eventually(func() error {
		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "po-val",
				Namespace: "default",
			},
			Immutable:  nil,
			Data:       map[string]string{"values.yaml": overrideData0},
			BinaryData: nil,
		}
		err := CreateConfigMap(&configMap)
		return err
	}, waitTimeOut, pollingInterval).Should(gomega.BeNil())

	t.Logs.Info("Update VZ with overrides and wait for Re-Install to complete")
	gomega.Eventually(func() bool {
		cr, err := pkg.GetVerrazzano()
		if err != nil {
			ginkgo.AbortSuite("Couldn't get the Verrazzano resource")
		}
		overrides := []vzapi.Overrides{
			{
				ConfigMapRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "po-val",
					},
					Key:      "values.yaml",
					Optional: nil,
				},
			},
		}
		watchValues := true
		cr.Spec.Components.PrometheusOperator.ValueOverrides = overrides
		cr.Spec.Components.PrometheusOperator.MonitorChanges = &watchValues
		UpdateVZ(cr)

		cr, err = pkg.GetVerrazzano()
		return cr.Status.State == "Ready"
	}, waitTimeOut, pollingInterval).Should(gomega.BeTrue())
	beforeSuitePassed = true
})

var failed = false
var beforeSuitePassed = false
var _ = t.AfterEach(func() {
	failed = failed || ginkgo.CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

var _ = t.Describe("Verify Override Post Upgrade", func() {
	t.It("Prometheus Operator pod should have the overriden label",
		gomega.Eventually(
			func() bool {
				pods, err := pkg.GetPodsFromSelector(&metav1.LabelSelector{
					MatchLabels: map[string]string{"override": "true"},
				}, constants.VerrazzanoMonitoringNamespace)

				if err != nil {
					ginkgo.AbortSuite(fmt.Sprintf("Label override not found with the given error: %v", err))
				}

				return len(pods) == 1
			}, waitTimeOut, pollingInterval).Should(gomega.BeTrue()))
})

func UpdateVZ(cr *vzapi.Verrazzano) {
	var err error
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		ginkgo.AbortSuite(fmt.Sprintf("Failed to get kubeconfig location with error: %v", err))
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		ginkgo.AbortSuite(fmt.Sprintf("Failed to get kubeconfig with error: %v", err))
	}

	vzClient, err := vpoClient.NewForConfig(config)
	if err != nil {
		ginkgo.AbortSuite(fmt.Sprintf("Failed to get verrazzano client with error: %v", err))
	}

	_, err = vzClient.VerrazzanoV1alpha1().Verrazzanos(cr.Namespace).Update(context.TODO(), cr, metav1.UpdateOptions{})
	if err != nil {
		ginkgo.AbortSuite(fmt.Sprintf("Failed to update Verrazzano with error: %v", err))
	}
}
