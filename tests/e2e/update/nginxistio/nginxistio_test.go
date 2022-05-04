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
	nginxName         = "controller"
	nginxLabel        = "app.kubernetes.io/component"
	istioIngressName  = "istio-egressgateway"
	istioIngressLabel = "app"
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
					Key:      "app",
					Operator: "In",
					Values: []string{
						istioIngressName,
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

	expectedNginxRunning := uint32(1)
	update.ValidatePods(nginxName, nginxLabel, constants.IngressNamespace, expectedNginxRunning, false)

	expectedIstioRunning := uint32(1)
	if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
		expectedIstioRunning = 2
	}
	update.ValidatePods(istioIngressName, istioIngressLabel, constants.IstioSystemNamespace, expectedIstioRunning, false)
})

var _ = t.Describe("Update nginx-istio", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-nginx-istio verify", Label("f:platform-lcm.nginx-istio-verify"), func() {
		t.It("nginx-istio default replicas", func() {
			cr := update.GetCR()

			expectedNginxRunning := uint32(1)
			update.ValidatePods(nginxName, nginxLabel, constants.IngressNamespace, expectedNginxRunning, false)

			expectedIstioRunning := uint32(1)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedIstioRunning = 2
			}
			update.ValidatePods(istioIngressName, istioIngressLabel, constants.IstioSystemNamespace, expectedIstioRunning, false)
		})
	})

	t.Describe("verrazzano-nginx-istio update", Label("f:platform-lcm.nginx-istio-update"), func() {
		t.It("nginx-istio update", func() {
			m := NginxAutoscalingIstioRelicasAffintyModifier{nginxReplicas: nodeCount, istioReplicas: nodeCount}
			update.UpdateCR(m)

			expectedNginxRunning := nodeCount
			update.ValidatePods(nginxName, nginxLabel, constants.IngressNamespace, expectedNginxRunning, false)

			expectedIstioRunning := nodeCount - 1
			expectedIstioPending := true
			if nodeCount == 1 {
				expectedIstioRunning = nodeCount
				expectedIstioPending = false
			}
			update.ValidatePods(istioIngressName, istioIngressLabel, constants.IstioSystemNamespace, expectedIstioRunning, expectedIstioPending)
		})
	})
})
