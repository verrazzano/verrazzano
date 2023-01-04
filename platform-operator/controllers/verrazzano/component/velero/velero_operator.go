// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	deploymentName         = "velero"
	veleroAwsPluginName    = "velero-plugin-for-aws"
	imagePullSecretHelmKey = "image.imagePullSecrets[0]"
)

// isVeleroOperatorReady checks if the Velero deployment is ready
func isVeleroOperatorReady(context spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix) &&
		ready.DaemonSetsAreReady(context.Log(), context.Client(), daemonSets, 1, componentPrefix)
}

// AppendOverrides appends Helm value overrides for the Velero component's Helm chart
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	arguments := []bom.KeyValue{
		{Key: "initContainers[0].name", Value: veleroAwsPluginName},
		{Key: "initContainers[0].imagePullPolicy", Value: "IfNotPresent"},
		{Key: "initContainers[0].volumeMounts[0].name", Value: "plugins"},
		{Key: "initContainers[0].volumeMounts[0].mountPath", Value: "/target"},
		{Key: "initContainers[0].securityContext.allowPrivilegeEscalation", Value: "false"},
		{Key: "initContainers[0].securityContext.capabilities.drop[0]", Value: "ALL"},
		{Key: "initContainers[0].securityContext.privileged", Value: "false"},
		{Key: "metrics.serviceMonitor.namespace", Value: ComponentNamespace},
	}
	kvs = append(kvs, arguments...)
	return kvs, nil
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Velero != nil {
			return effectiveCR.Spec.Components.Velero.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Velero != nil {
			return effectiveCR.Spec.Components.Velero.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
func ensureVeleroNamespace(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Creating namespace %s for Velero.", ComponentNamespace)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	namespace.Labels["verrazzano.io/namespace"] = ComponentNamespace
	istio := ctx.EffectiveCR().Spec.Components.Istio
	if istio != nil && istio.IsInjectionEnabled() {
		namespace.Labels["istio-injection"] = "enabled"
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctrlerrors.RetryableError{Source: ComponentName, Cause: err}
	}
	return nil
}
