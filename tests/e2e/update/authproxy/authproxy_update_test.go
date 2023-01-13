// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	"sigs.k8s.io/yaml"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	authProxyLabelValue = "verrazzano-authproxy"
	authProxyLabelKey   = "app"
	explicitReplicas    = uint32(3)
	waitTimeout         = 20 * time.Minute
	pollingInterval     = 10 * time.Second
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
	t.Logs.Debugf("AuthProxyReplicasModifier CR: %v", marshalCRToString(cr.Spec))
}

func (u AuthProxyReplicasModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
	}
	authProxyReplicasOverridesYaml := fmt.Sprintf(`replicas: %v`, u.replicas)
	cr.Spec.Components.AuthProxy.ValueOverrides = pkg.CreateOverridesOrDie(authProxyReplicasOverridesYaml)
	t.Logs.Debugf("AuthProxyReplicasModifierV1beta1 CR: %s", marshalCRToString(cr.Spec))
}

func (u AuthProxyPodPerNodeAffintyModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
	}
	authProxyAffinityOverridesYaml := fmt.Sprintf(`affinity: |
              podAntiAffinity:
                preferredDuringSchedulingIgnoredDuringExecution:
                - podAffinityTerm:
                    labelSelector:
                      matchExpressions:
                      - key: %v
                        operator: In
                        values:
                        - %v
                    topologyKey: kubernetes.io/hostname
                  weight: 100`, authProxyLabelKey, authProxyLabelValue)
	cr.Spec.Components.AuthProxy.ValueOverrides = pkg.CreateOverridesOrDie(authProxyAffinityOverridesYaml)
	t.Logs.Debugf("AuthProxyPodPerNodeAffintyModifierV1beta1 CR: %v", marshalCRToString(cr.Spec))
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
	t.Logs.Debugf("AuthProxyPodPerNodeAffintyModifier CR: %v", marshalCRToString(cr.Spec))
}

func (u AuthProxyDefaultModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{}
	t.Logs.Debugf("AuthProxyDefaultModifier CR: %v", cr.Spec)
}

func (u AuthProxyDefaultModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
	t.Logs.Debugf("AuthProxyDefaultModifierV1beta1 CR: %v", marshalCRToString(cr.Spec))
}

var t = framework.NewTestFramework("update authproxy")

var nodeCount uint32

var beforeSuite = t.BeforeSuiteFunc(func() {
	var err error
	nodeCount, err = pkg.GetSchedulableNodeCount()
	if err != nil {
		Fail(err.Error())
	}
	t.Logs.Info("Schedulable nodes: %v", nodeCount)
})

var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	m := AuthProxyDefaultModifier{}
	update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, uint32(1), false)
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Update authProxy", Label("f:platform-lcm.update"), func() {
	t.Describe("verrazzano-authproxy verify", Label("f:platform-lcm.authproxy-verify"), func() {
		t.It("authproxy default replicas", func() {
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, uint32(1), false)
		})
	})

	t.Describe("verrazzano-authproxy update replicas", Label("f:platform-lcm.authproxy-update-replicas"), func() {
		t.It("authproxy explicit replicas v1alpha1", func() {
			m := AuthProxyReplicasModifier{replicas: nodeCount}
			update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
			expectedRunning := nodeCount
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

	t.Describe("verrazzano-authproxy update affinity", Label("f:platform-lcm.authproxy-update-affinity"), func() {
		t.It("authproxy explicit affinity v1alpha1", func() {
			m := AuthProxyPodPerNodeAffintyModifier{}
			update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)

			expectedRunning := nodeCount
			expectedPending := true
			if nodeCount == 1 {
				expectedPending = false
			}
			t.Logs.Debugf("Expected running: %v, expected pending %v", expectedRunning, expectedPending)
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, expectedPending)
		})
	})

	t.Describe("verrazzano-authproxy update replicas with v1beta1 client", Label("f:platform-lcm.authproxy-update-replicas"), func() {
		t.It("authproxy explicit replicas v1beta1", func() {
			m := AuthProxyReplicasModifierV1beta1{replicas: explicitReplicas}
			update.UpdateCRV1beta1WithRetries(m, pollingInterval, waitTimeout)
			expectedRunning := explicitReplicas
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

	t.Describe("verrazzano-authproxy update affinity using v1beta1 client", Label("f:platform-lcm.authproxy-update-affinity"), func() {
		t.It("authproxy explicit affinity v1beta1", func() {
			m := AuthProxyPodPerNodeAffintyModifierV1beta1{}
			update.UpdateCRV1beta1WithRetries(m, pollingInterval, waitTimeout)
			expectedRunning := explicitReplicas
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
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
