// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginxistio

import (
	"fmt"
	"os"
	"strconv"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"

	"github.com/verrazzano/verrazzano/pkg/constants"
)

const (
	nginxLabelValue        = "controller"
	nginxLabelKey          = "app.kubernetes.io/component"
	istioIngressLabelValue = "istio-egressgateway"
	istioIngressLabelKey   = "app"
)

type NginxAutoscalingIstioRelicasAffintyModifier struct {
	nginxReplicas uint32
	istioReplicas uint32
}

type NginxIstioDefaultModifier struct {
}

func (m NginxAutoscalingIstioRelicasAffintyModifier) ModifyCR(cr *vzapi.Verrazzano) {
	// update nginx
	if cr.Spec.Components.Ingress == nil {
		cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	}
	cr.Spec.Components.Ingress.NGINXInstallArgs = []vzapi.InstallArgs{
		{
			Name:  "controller.autoscaling.enabled",
			Value: "true",
		},
		{
			Name:  "controller.autoscaling.minReplicas",
			Value: fmt.Sprint(m.nginxReplicas),
		},
	}
	// update istio
	if cr.Spec.Components.Istio == nil {
		cr.Spec.Components.Istio = &vzapi.IstioComponent{}
	}
	if cr.Spec.Components.Istio.Ingress == nil {
		cr.Spec.Components.Istio.Ingress = &vzapi.IstioIngressSection{}
	}
	if cr.Spec.Components.Istio.Ingress.Kubernetes == nil {
		cr.Spec.Components.Istio.Ingress.Kubernetes = &vzapi.IstioKubernetesSection{}
	}
	cr.Spec.Components.Istio.Ingress.Kubernetes.Replicas = m.istioReplicas
	if cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity == nil {
		cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity = &corev1.Affinity{}
	}
	if cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity == nil {
		cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	list := cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	list = append(list, corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: nil,
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      istioIngressLabelKey,
					Operator: "In",
					Values: []string{
						istioIngressLabelValue,
					},
				},
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	})
	cr.Spec.Components.Istio.Ingress.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = list
}

func (u NginxIstioDefaultModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{}
	cr.Spec.Components.Istio = &vzapi.IstioComponent{}
}

var t = framework.NewTestFramework("update nginx-istio")

var nodeCount uint32 = 1

var _ = t.BeforeSuite(func() {
	kindNodeCount := os.Getenv("KIND_NODE_COUNT")
	if len(kindNodeCount) > 0 {
		u, err := strconv.ParseUint(kindNodeCount, 10, 32)
		if err == nil {
			nodeCount = uint32(u)
		}
	}
})

var _ = t.AfterSuite(func() {
	m := NginxIstioDefaultModifier{}
	update.UpdateCR(m)
	cr := update.GetCR()

	update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, uint32(1), false)

	expectedIstioRunning := uint32(1)
	if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
		expectedIstioRunning = 2
	}
	update.ValidatePods(istioIngressLabelValue, istioIngressLabelKey, constants.IstioSystemNamespace, expectedIstioRunning, false)
})

var _ = t.Describe("Update nginx-istio", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-nginx-istio verify", Label("f:platform-lcm.nginx-istio-verify"), func() {
		t.It("nginx-istio default replicas", func() {
			cr := update.GetCR()

			update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, uint32(1), false)

			expectedIstioRunning := uint32(1)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedIstioRunning = 2
			}
			update.ValidatePods(istioIngressLabelValue, istioIngressLabelKey, constants.IstioSystemNamespace, expectedIstioRunning, false)
		})
	})

	t.Describe("verrazzano-nginx-istio update", Label("f:platform-lcm.nginx-istio-update"), func() {
		t.It("nginx-istio update", func() {
			istioCount := nodeCount - 1
			if nodeCount == 1 {
				istioCount = nodeCount
			}
			m := NginxAutoscalingIstioRelicasAffintyModifier{nginxReplicas: nodeCount, istioReplicas: istioCount}
			update.UpdateCR(m)

			update.ValidatePods(nginxLabelValue, nginxLabelKey, constants.IngressNamespace, nodeCount, false)

			update.ValidatePods(istioIngressLabelValue, istioIngressLabelKey, constants.IstioSystemNamespace, istioCount, false)
		})
	})
})
