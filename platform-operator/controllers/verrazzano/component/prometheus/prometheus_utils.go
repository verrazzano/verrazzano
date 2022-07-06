// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutateVerrazzanoMonitoringNamespace modifies the given namespace for the Monitoring subcomponents
// with the appropriate labels, in one location. If the provided namespace is not the Verrazzano
// monitoring namespace, it is ignored.
func MutateVerrazzanoMonitoringNamespace(ctx spi.ComponentContext, namespace *corev1.Namespace) {
	if namespace.Name != vpoconst.VerrazzanoMonitoringNamespace {
		return
	}
	if namespace.Labels == nil {
		namespace.Labels = map[string]string{}
	}
	namespace.Labels[v8oconst.LabelVerrazzanoNamespace] = vpoconst.VerrazzanoMonitoringNamespace

	istio := ctx.EffectiveCR().Spec.Components.Istio
	if istio != nil && istio.IsInjectionEnabled() {
		namespace.Labels[v8oconst.LabelIstioInjection] = "enabled"
	}
}

// GetVerrazzanoMonitoringNamespace creates and returns a namespace object for the Monitoring
// subcomponents' namespace
func GetVerrazzanoMonitoringNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: vpoconst.VerrazzanoMonitoringNamespace,
		},
	}
}
