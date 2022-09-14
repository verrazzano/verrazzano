// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	v1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	authProxyLabelValue = "verrazzano-authproxy"
	authProxyLabelKey   = "app"
)

type AuthProxyReplicasModifier struct {
	replicas uint32
}

type AuthProxyReplicasModifierV1beta1 struct {
	replicas uint32
}

type AuthProxyPodPerNodeAffintyModifierV1beta1 struct {
}

type AuthProxyPodPerNodeAffintyModifier struct {
}

type AuthProxyDefaultModifier struct {
}

type AuthProxyDefaultModifierV1beta1 struct {
}

func (u AuthProxyReplicasModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes == nil {
		cr.Spec.Components.AuthProxy.Kubernetes = &vzapi.AuthProxyKubernetesSection{}
	}
	cr.Spec.Components.AuthProxy.Kubernetes.Replicas = u.replicas
}

func (u AuthProxyReplicasModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes == nil {
		cr.Spec.Components.AuthProxy.Kubernetes = &v1beta1.AuthProxyKubernetesSection{}
	}
	cr.Spec.Components.AuthProxy.Kubernetes.Replicas = u.replicas
}

func (u AuthProxyPodPerNodeAffintyModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes == nil {
		cr.Spec.Components.AuthProxy.Kubernetes = &v1beta1.AuthProxyKubernetesSection{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes.Affinity == nil {
		cr.Spec.Components.AuthProxy.Kubernetes.Affinity = &corev1.Affinity{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity == nil {
		cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	list := cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	list = append(list, corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: nil,
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      authProxyLabelKey,
					Operator: "In",
					Values: []string{
						authProxyLabelValue,
					},
				},
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	})
	cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = list
}

func (u AuthProxyPodPerNodeAffintyModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes == nil {
		cr.Spec.Components.AuthProxy.Kubernetes = &vzapi.AuthProxyKubernetesSection{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes.Affinity == nil {
		cr.Spec.Components.AuthProxy.Kubernetes.Affinity = &corev1.Affinity{}
	}
	if cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity == nil {
		cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	list := cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	list = append(list, corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: nil,
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      authProxyLabelKey,
					Operator: "In",
					Values: []string{
						authProxyLabelValue,
					},
				},
			},
		},
		TopologyKey: "kubernetes.io/hostname",
	})
	cr.Spec.Components.AuthProxy.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = list
}

func (u AuthProxyDefaultModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
}

func (u AuthProxyDefaultModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
}

var t = framework.NewTestFramework("update authproxy")

var nodeCount uint32

var _ = t.BeforeSuite(func() {
	var err error
	nodeCount, err = pkg.GetNodeCount()
	if err != nil {
		Fail(err.Error())
	}
})

var _ = t.AfterSuite(func() {
	m := AuthProxyDefaultModifier{}
	err := update.UpdateCR(m)
	if err != nil {
		Fail(err.Error())
	}

	cr := update.GetCR()
	expectedRunning := uint32(1)
	if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
		expectedRunning = 2
	}
	update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
})

var _ = t.Describe("Update authProxy", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-authproxy verify", Label("f:platform-lcm.authproxy-verify"), func() {
		t.It("authproxy default replicas", func() {
			cr := update.GetCR()

			expectedRunning := uint32(1)
			if cr.Spec.Profile == "prod" || cr.Spec.Profile == "" {
				expectedRunning = 2
			}
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

	t.Describe("verrazzano-authproxy update replicas", Label("f:platform-lcm.authproxy-update-replicas"), func() {
		t.It("authproxy explicit replicas", func() {
			m := AuthProxyReplicasModifier{replicas: nodeCount}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			expectedRunning := nodeCount
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

	t.Describe("verrazzano-authproxy update replicas with v1beta1 client", Label("f:platform-lcm.authproxy-update-replicas"), func() {
		t.It("authproxy explicit replicas", func() {
			m := AuthProxyReplicasModifierV1beta1{replicas: nodeCount}
			err := update.UpdateCRV1beta1(m)
			if err != nil {
				Fail(err.Error())
			}

			expectedRunning := nodeCount
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

	t.Describe("verrazzano-authproxy update affinity", Label("f:platform-lcm.authproxy-update-affinity"), func() {
		t.It("authproxy explicit affinity", func() {
			m := AuthProxyPodPerNodeAffintyModifier{}
			err := update.UpdateCR(m)
			if err != nil {
				Fail(err.Error())
			}

			expectedRunning := nodeCount - 1
			expectedPending := true
			if nodeCount == 1 {
				expectedRunning = nodeCount
				expectedPending = false
			}
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
		})
	})
})
