// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"testing"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSyncer_updatePrometheusMonitorsClusterName(t *testing.T) {
	type fields struct {
		OldManagedClusterName string
		NewManagedClusterName string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{"managed cluster name changed", fields{OldManagedClusterName: "local", NewManagedClusterName: "mgdcluster1"}},
		{"managed cluster name unchanged", fields{OldManagedClusterName: "mgcluster", NewManagedClusterName: "mgcluster"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns1 := "ns1"
			ns2 := "ns2"
			ns3 := "ns3"
			smWithOldClusterName := createTestServiceMonitor(true, tt.fields.OldManagedClusterName, "smold", ns1)
			smWithNewClusterName := createTestServiceMonitor(true, tt.fields.NewManagedClusterName, "smnew", ns2)
			smNoClusterName := createTestServiceMonitor(false, "", "smnone", ns3)
			pmWithOldClusterName := createTestPodMonitor(true, tt.fields.OldManagedClusterName, "pmold", ns1)
			pmWithNewClusterName := createTestPodMonitor(true, tt.fields.NewManagedClusterName, "pmnew", ns3)
			pmNoClusterName := createTestPodMonitor(false, "", "pmnone", ns2)

			scheme := runtime.NewScheme()
			_ = promoperapi.AddToScheme(scheme)
			mgdClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
				smWithOldClusterName, smWithNewClusterName, smNoClusterName,
				pmWithOldClusterName, pmWithNewClusterName, pmNoClusterName).Build()
			adminClient := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

			s := &Syncer{
				AdminClient:        adminClient,
				LocalClient:        mgdClient,
				Log:                zap.S().With(tt.name),
				ManagedClusterName: tt.fields.NewManagedClusterName,
				Context:            context.TODO(),
			}
			err := s.updatePrometheusMonitorsClusterName()
			assert.NoError(t, err)
			assertServiceMonitorLabel(t, mgdClient, smWithOldClusterName, tt.fields.NewManagedClusterName)
			assertServiceMonitorLabel(t, mgdClient, smWithNewClusterName, tt.fields.NewManagedClusterName)
			assertPodMonitorLabel(t, mgdClient, pmWithOldClusterName, tt.fields.NewManagedClusterName)
			assertPodMonitorLabel(t, mgdClient, pmWithNewClusterName, tt.fields.NewManagedClusterName)
		})
	}
}

func assertServiceMonitorLabel(t *testing.T, client client.WithWatch, sm *promoperapi.ServiceMonitor, newClusterName string) {
	retrievedSM := promoperapi.ServiceMonitor{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: sm.Namespace, Name: sm.Name}, &retrievedSM)
	assert.NoError(t, err)
	for i, ep := range retrievedSM.Spec.Endpoints {
		assertRCLabels(t, sm.Spec.Endpoints[i].RelabelConfigs, ep.RelabelConfigs, newClusterName)
	}
}

func assertPodMonitorLabel(t *testing.T, client client.WithWatch, pm *promoperapi.PodMonitor, newClusterName string) {
	retrievedPM := promoperapi.PodMonitor{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: pm.Namespace, Name: pm.Name}, &retrievedPM)
	assert.NoError(t, err)
	assert.Equal(t, len(pm.Spec.PodMetricsEndpoints), len(retrievedPM.Spec.PodMetricsEndpoints))
	for i, ep := range retrievedPM.Spec.PodMetricsEndpoints {
		assertRCLabels(t, pm.Spec.PodMetricsEndpoints[i].RelabelConfigs, ep.RelabelConfigs, newClusterName)
	}
}

func assertRCLabels(t *testing.T, oldRCs []*promoperapi.RelabelConfig, newRCs []*promoperapi.RelabelConfig, clusterName string) {
	assert.Equal(t, len(oldRCs), len(newRCs))
	for _, rc := range newRCs {
		if rc.TargetLabel == prometheusClusterNameLabel {
			assert.Equal(t, clusterName, rc.Replacement)
		}
	}
}

func createTestServiceMonitor(hasClusterNameRelabelConfig bool, clusterName string, monitorName string, monitorNS string) *promoperapi.ServiceMonitor {
	relabelConfigs := []*promoperapi.RelabelConfig{}
	if hasClusterNameRelabelConfig {
		relabelConfigs = append(relabelConfigs, &promoperapi.RelabelConfig{TargetLabel: prometheusClusterNameLabel, Replacement: clusterName})
	}
	return &promoperapi.ServiceMonitor{
		ObjectMeta: v12.ObjectMeta{Name: monitorName, Namespace: monitorNS},
		Spec: promoperapi.ServiceMonitorSpec{
			Endpoints: []promoperapi.Endpoint{
				{RelabelConfigs: relabelConfigs},
			},
		}}
}

func createTestPodMonitor(hasClusterNameRelabelConfig bool, clusterName string, monitorName string, monitorNS string) *promoperapi.PodMonitor {
	relabelConfigs := []*promoperapi.RelabelConfig{}
	if hasClusterNameRelabelConfig {
		relabelConfigs = append(relabelConfigs, &promoperapi.RelabelConfig{TargetLabel: prometheusClusterNameLabel, Replacement: clusterName})
	}
	return &promoperapi.PodMonitor{
		ObjectMeta: v12.ObjectMeta{Name: monitorName, Namespace: monitorNS},
		Spec: promoperapi.PodMonitorSpec{
			PodMetricsEndpoints: []promoperapi.PodMetricsEndpoint{
				{RelabelConfigs: relabelConfigs},
			},
		}}
}
