// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nodeexporter

import (
	"context"
	"fmt"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	daemonsetName     = "prometheus-node-exporter" // Should match fullName override in prometheus-node-exporter-values.yaml
	networkPolicyName = "node-exporter"
)

// isPrometheusNodeExporterReady checks if the Prometheus Node-Exporter daemonset is ready
func (c prometheusNodeExporterComponent) isPrometheusNodeExporterReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DaemonSetsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DaemonsetNames, 1, prefix)
}

// PreInstall implementation for the Prometheus Node-Exporter Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Node-Exporter preInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Prometheus Node-Exporter", ComponentNamespace)
	ns := common.GetVerrazzanoMonitoringNamespace()
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), ns, func() error {
		common.MutateVerrazzanoMonitoringNamespace(ctx, ns)
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// GetOverrides returns install overrides for a component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.PrometheusNodeExporter != nil {
			return effectiveCR.Spec.Components.PrometheusNodeExporter.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.PrometheusNodeExporter != nil {
			return effectiveCR.Spec.Components.PrometheusNodeExporter.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// createOrUpdateNetworkPolicies creates or updates network policies for this component
func createOrUpdateNetworkPolicies(ctx spi.ComponentContext) error {
	netPolicy := &netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: networkPolicyName, Namespace: ComponentNamespace}}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client(), netPolicy, func() error {
		netPolicy.Spec = newNetworkPolicySpec()
		return nil
	})

	return err
}

// newNetworkPolicy returns a populated NetworkPolicySpec with ingress rules for Prometheus node exporter
func newNetworkPolicySpec() netv1.NetworkPolicySpec {
	tcpProtocol := corev1.ProtocolTCP
	port := intstr.FromInt(9100)

	return netv1.NetworkPolicySpec{
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "prometheus-node-exporter",
			},
		},
		PolicyTypes: []netv1.PolicyType{
			netv1.PolicyTypeIngress,
		},
		Ingress: []netv1.NetworkPolicyIngressRule{
			{
				// allow ingress to port 9100 from Prometheus
				From: []netv1.NetworkPolicyPeer{
					{
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app.kubernetes.io/name": "prometheus",
							},
						},
					},
				},
				Ports: []netv1.NetworkPolicyPort{
					{
						Protocol: &tcpProtocol,
						Port:     &port,
					},
				},
			},
		},
	}
}
