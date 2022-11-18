// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nodeexporter

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var testScheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = vzapi.AddToScheme(testScheme)
}

// TestIsPrometheusNodeExporterReady tests the isPrometheusNodeExporterReady function for the Prometheus Node-Exporter
func TestIsPrometheusNodeExporterReady(t *testing.T) {
	tests := []struct {
		name       string
		client     client.Client
		expectTrue bool
	}{
		{
			// GIVEN the Prometheus Node-Exporter deployment exists and there are available replicas
			// WHEN we call isPrometheusNodeExporterReady
			// THEN the call returns true
			name: "Test IsReady when Prometheus Node-Exporter is successfully deployed",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      daemonsetName,
						Labels:    map[string]string{"app": "node-exporter"},
					},
					Spec: appsv1.DaemonSetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "node-exporter"},
						},
					},
					Status: appsv1.DaemonSetStatus{
						NumberAvailable:        1,
						UpdatedNumberScheduled: 1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      ComponentName,
						Labels: map[string]string{
							"app":                      "node-exporter",
							"controller-revision-hash": "test-95d8c5d96",
						},
					},
				},
				&appsv1.ControllerRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      ComponentName + "-test-95d8c5d96",
						Namespace: ComponentNamespace,
					},
					Revision: 1,
				},
			).Build(),
			expectTrue: true,
		},
		{
			// GIVEN the Prometheus Node-Exporter deployment exists and there are no available replicas
			// WHEN we call isPrometheusNodeExporterReady
			// THEN the call returns false
			name: "Test IsReady when Prometheus Node-Exporter deployment is not ready",
			client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      daemonsetName,
					},
					Status: appsv1.DaemonSetStatus{
						NumberAvailable:        0,
						UpdatedNumberScheduled: 1,
					},
				},
			).Build(),
			expectTrue: false,
		},
		{
			// GIVEN the Prometheus Node-Exporter deployment does not exist
			// WHEN we call isPrometheusNodeExporterReady
			// THEN the call returns false
			name:       "Test IsReady when Prometheus Node-Exporter deployment does not exist",
			client:     fake.NewClientBuilder().WithScheme(testScheme).Build(),
			expectTrue: false,
		},
	}
	nodeExporter := NewComponent().(prometheusNodeExporterComponent)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, nil, false)
			assert.Equal(t, tt.expectTrue, nodeExporter.isPrometheusNodeExporterReady(ctx))
		})
	}
}

// TestCreateOrUpdateNetworkPolicies tests the createOrUpdateNetworkPolicies function
func TestCreateOrUpdateNetworkPolicies(t *testing.T) {
	// GIVEN a Prometheus Node Exporter component
	// WHEN  the createOrUpdateNetworkPolicies function is called
	// THEN  no error is returned and the expected network policies have been created
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	ctx := spi.NewFakeContext(fakeClient, &vzapi.Verrazzano{}, nil, false)

	err := createOrUpdateNetworkPolicies(ctx)
	assert.NoError(t, err)

	netPolicy := &netv1.NetworkPolicy{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: networkPolicyName, Namespace: ComponentNamespace}, netPolicy)
	assert.NoError(t, err)
	assert.Equal(t, []netv1.PolicyType{netv1.PolicyTypeIngress}, netPolicy.Spec.PolicyTypes)
	assert.Equal(t, int32(9100), netPolicy.Spec.Ingress[0].Ports[0].Port.IntVal)
}

// test preinstall when dryrun is false
func TestPreInstall(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, false)
	assert.Nil(t, preInstall(ctx))
}

// test preinstall when dryrun is true
func TestPreInstallDryRun(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	ctx := spi.NewFakeContext(c, &vzapi.Verrazzano{}, nil, true)
	assert.Nil(t, preInstall(ctx))
}

// test GetOverrides method
func TestGetOverrides(t *testing.T) {
	ref := &corev1.ConfigMapKeySelector{
		Key: "foo",
	}
	o := v1beta1.InstallOverrides{
		ValueOverrides: []v1beta1.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	oV1Alpha1 := vzapi.InstallOverrides{
		ValueOverrides: []vzapi.Overrides{
			{
				ConfigMapRef: ref,
			},
		},
	}
	var tests = []struct {
		name string
		cr   runtime.Object
		res  interface{}
	}{
		{
			"overrides when component not nil, v1alpha1",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						PrometheusNodeExporter: &vzapi.PrometheusNodeExporterComponent{
							InstallOverrides: oV1Alpha1,
						},
					},
				},
			},
			oV1Alpha1.ValueOverrides,
		},
		{
			"Empty overrides when component nil",
			&v1beta1.Verrazzano{},
			[]v1beta1.Overrides{},
		},
		{
			"overrides when component not nil",
			&v1beta1.Verrazzano{
				Spec: v1beta1.VerrazzanoSpec{
					Components: v1beta1.ComponentSpec{
						PrometheusNodeExporter: &v1beta1.PrometheusNodeExporterComponent{
							InstallOverrides: o,
						},
					},
				},
			},
			o.ValueOverrides,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := GetOverrides(tt.cr)
			assert.EqualValues(t, tt.res, override)
		})
	}
}
