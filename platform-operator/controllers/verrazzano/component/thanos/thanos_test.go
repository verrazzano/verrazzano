// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"context"
	"fmt"
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	istioclisec "istio.io/client-go/pkg/apis/security/v1beta1"
	netv1 "k8s.io/api/networking/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const bomFilePathOverride = "../../../../verrazzano-bom.json"

// TestGetOverrides tests if Thanos overrides are properly collected
// GIVEN a call to GetOverrides
// WHEN the VZ CR Thanos component has overrides
// THEN the overrides are returned from this function
func TestGetOverrides(t *testing.T) {
	testKey := "test-key"
	testVal := "test-val"
	jsonVal := []byte(fmt.Sprintf("{\"%s\":\"%s\"}", testKey, testVal))

	vzA1CR := &v1alpha1.Verrazzano{}
	vzA1CROverrides := vzA1CR.DeepCopy()
	vzA1CROverrides.Spec.Components.Thanos = &v1alpha1.ThanosComponent{
		InstallOverrides: v1alpha1.InstallOverrides{
			ValueOverrides: []v1alpha1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	vzB1CR := &v1beta1.Verrazzano{}
	vzB1CROverrides := vzB1CR.DeepCopy()
	vzB1CROverrides.Spec.Components.Thanos = &v1beta1.ThanosComponent{
		InstallOverrides: v1beta1.InstallOverrides{
			ValueOverrides: []v1beta1.Overrides{
				{
					Values: &apiextensionsv1.JSON{
						Raw: jsonVal,
					},
				},
			},
		},
	}

	tests := []struct {
		name           string
		verrazzanoA1   *v1alpha1.Verrazzano
		verrazzanoB1   *v1beta1.Verrazzano
		expA1Overrides interface{}
		expB1Overrides interface{}
	}{
		{
			name:           "test no overrides",
			verrazzanoA1:   vzA1CR,
			verrazzanoB1:   vzB1CR,
			expA1Overrides: []v1alpha1.Overrides{},
			expB1Overrides: []v1beta1.Overrides{},
		},
		{
			name:           "test v1alpha1 enabled nil",
			verrazzanoA1:   vzA1CROverrides,
			verrazzanoB1:   vzB1CROverrides,
			expA1Overrides: vzA1CROverrides.Spec.Components.Thanos.ValueOverrides,
			expB1Overrides: vzB1CROverrides.Spec.Components.Thanos.ValueOverrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asserts.Equal(t, tt.expA1Overrides, NewComponent().GetOverrides(tt.verrazzanoA1))
			asserts.Equal(t, tt.expB1Overrides, NewComponent().GetOverrides(tt.verrazzanoB1))
		})
	}
}

// TestAppendOverrides tests if Thanos overrides are appendded
// GIVEN a call to AppendOverrides
// WHEN the bom is populated with the Thanos image
// THEN the overrides are returned from this function
func TestAppendOverrides(t *testing.T) {
	config.SetDefaultBomFilePath(bomFilePathOverride)
	kvs, err := AppendOverrides(nil, "", "", "", []bom.KeyValue{})
	asserts.NoError(t, err)

	expectedKVS := map[string]string{
		"image.registry":   "ghcr.io",
		"image.repository": "verrazzano/thanos",
	}
	for _, kv := range kvs {
		if kv.Key == "image.tag" {
			asserts.NotEmpty(t, kv.Value)
			continue
		}
		val, ok := expectedKVS[kv.Key]
		asserts.True(t, ok)
		asserts.Equal(t, val, kv.Value)
	}
}

// TestCreateOrUpdateNetworkPolicies tests the createOrUpdateNetworkPolicies function
// GIVEN a Thanos component
// WHEN  the createOrUpdateNetworkPolicies function is called
// THEN  no error is returned and the expected network policies have been created
func TestCreateOrUpdateNetworkPolicies(t *testing.T) {

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)

	err := createOrUpdateNetworkPolicies(ctx)
	asserts.NoError(t, err)

	netPolicy := &netv1.NetworkPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: thanosNetPolicyName, Namespace: ComponentNamespace}, netPolicy)
	asserts.NoError(t, err)
	asserts.Len(t, netPolicy.Spec.Ingress, 1)
	asserts.Equal(t, []netv1.PolicyType{netv1.PolicyTypeIngress}, netPolicy.Spec.PolicyTypes)
	asserts.Equal(t, int32(10902), netPolicy.Spec.Ingress[0].Ports[0].Port.IntVal)
	asserts.Contains(t, netPolicy.Spec.Ingress[0].From[0].PodSelector.MatchExpressions[0].Values, "verrazzano-authproxy")
}

// TestCreateOrUpdatePrometheusAuthPolicy tests the createOrUpdatePrometheusAuthPolicy function
func TestCreateOrUpdatePrometheusAuthPolicy(t *testing.T) {
	assertions := asserts.New(t)
	falseValue := false

	// GIVEN Thanos is being installed or upgraded
	// WHEN  we call the createOrUpdateComponentAuthPolicy function
	// THEN  the expected Istio authorization policy is created
	scheme := k8scheme.Scheme
	err := istioclisec.AddToScheme(scheme)
	assertions.NoError(err)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{}, nil, false)

	err = createOrUpdateComponentAuthPolicy(ctx)
	assertions.NoError(err)

	authPolicy := &istioclisec.AuthorizationPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: thanosAuthPolicyName}, authPolicy)
	assertions.NoError(err)

	assertions.Len(authPolicy.Spec.Rules, 1)
	assertions.Contains(authPolicy.Spec.Rules[0].From[0].Source.Principals, "cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy")

	// GIVEN Thanos is being installed or upgraded
	// AND   Istio is disabled
	// WHEN  we call the createOrUpdateComponentAuthPolicy function
	// THEN  no Istio authorization policy is created
	client = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	vz := &v1alpha1.Verrazzano{
		Spec: v1alpha1.VerrazzanoSpec{
			Components: v1alpha1.ComponentSpec{
				Istio: &v1alpha1.IstioComponent{
					Enabled: &falseValue,
				},
			},
		},
	}
	ctx = spi.NewFakeContext(client, vz, nil, false)

	err = createOrUpdateComponentAuthPolicy(ctx)
	assertions.NoError(err)

	authPolicy = &istioclisec.AuthorizationPolicy{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: thanosAuthPolicyName}, authPolicy)
	assertions.ErrorContains(err, "not found")
}
