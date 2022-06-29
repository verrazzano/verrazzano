// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nodeexporter

import (
	corev1 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
			client: fake.NewFakeClientWithScheme(testScheme,
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      daemonsetName,
						Labels:    map[string]string{"app": "test"},
					},
					Spec: appsv1.DaemonSetSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test"},
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
							"app":                      "test",
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
			),
			expectTrue: true,
		},
		{
			// GIVEN the Prometheus Node-Exporter deployment exists and there are no available replicas
			// WHEN we call isPrometheusNodeExporterReady
			// THEN the call returns false
			name: "Test IsReady when Prometheus Node-Exporter deployment is not ready",
			client: fake.NewFakeClientWithScheme(testScheme,
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ComponentNamespace,
						Name:      daemonsetName,
					},
					Status: appsv1.DaemonSetStatus{
						NumberAvailable:        0,
						UpdatedNumberScheduled: 1,
					},
				}),
			expectTrue: false,
		},
		{
			// GIVEN the Prometheus Node-Exporter deployment does not exist
			// WHEN we call isPrometheusNodeExporterReady
			// THEN the call returns false
			name:       "Test IsReady when Prometheus Node-Exporter deployment does not exist",
			client:     fake.NewFakeClientWithScheme(testScheme),
			expectTrue: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := spi.NewFakeContext(tt.client, &vzapi.Verrazzano{}, false)
			assert.Equal(t, tt.expectTrue, isPrometheusNodeExporterReady(ctx))
		})
	}
}
