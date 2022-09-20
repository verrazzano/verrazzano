// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package authproxy

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	authProxyLabelValue = "verrazzano-authproxy"
	authProxyLabelKey   = "app"
	explicitReplicas    = uint32(3)
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
	authProxyReplicasOverridesYaml := fmt.Sprintf(`replicas: %v`, u.replicas)
	cr.Spec.Components.AuthProxy.ValueOverrides = createOverridesOrDie(authProxyReplicasOverridesYaml)
}

func createOverridesOrDie(yamlString string) []v1beta1.Overrides {
	data, err := yaml.YAMLToJSON([]byte(yamlString))
	if err != nil {
		t.Logs.Errorf("Failed to convert yaml to JSON: %s", yamlString)
		panic(err)
	}
	return []v1beta1.Overrides{
		{
			ConfigMapRef: nil,
			SecretRef:    nil,
			Values: &apiextensionsv1.JSON{
				Raw: data,
			},
		},
	}
}

func (u AuthProxyPodPerNodeAffintyModifierV1beta1) ModifyCRV1beta1(cr *v1beta1.Verrazzano) {
	if cr.Spec.Components.AuthProxy == nil {
		cr.Spec.Components.AuthProxy = &v1beta1.AuthProxyComponent{}
	}
	authProxyAffinityOverridesYaml := fmt.Sprintf(`affinity:
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
	cr.Spec.Components.AuthProxy.ValueOverrides = createOverridesOrDie(authProxyAffinityOverridesYaml)
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
	m := AuthProxyDefaultModifierV1beta1{}
	err := update.UpdateCRV1beta1(m)
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
	/*	t.Describe("verrazzano-authproxy verify", Label("f:platform-lcm.authproxy-verify"), func() {
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
		})*/

	t.Describe("verrazzano-authproxy update replicas with v1beta1 client", Label("f:platform-lcm.authproxy-update-replicas"), func() {
		t.It("authproxy explicit replicas", func() {
			m := AuthProxyReplicasModifierV1beta1{replicas: explicitReplicas}
			err := update.UpdateCRV1beta1(m)
			if err != nil {
				Fail(err.Error())
			}
			expectedRunning := explicitReplicas
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

	t.Describe("verrazzano-authproxy update affinity using v1beta1 client", Label("f:platform-lcm.authproxy-update-affinity"), func() {
		t.It("authproxy explicit affinity", func() {
			m := AuthProxyPodPerNodeAffintyModifierV1beta1{}
			err := update.UpdateCRV1beta1(m)
			if err != nil {
				Fail(err.Error())
			}
			expectedRunning := explicitReplicas
			update.ValidatePods(authProxyLabelValue, authProxyLabelKey, constants.VerrazzanoSystemNamespace, expectedRunning, false)
		})
	})

})
