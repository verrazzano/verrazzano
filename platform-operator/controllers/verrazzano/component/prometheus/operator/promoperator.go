// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus"
	"strconv"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const deploymentName = "prometheus-operator-kube-p-operator"

// isPrometheusOperatorReady checks if the Prometheus operator deployment is ready
func isPrometheusOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Prometheus Operator Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Operator PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Prometheus Operator", ComponentNamespace)
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ComponentNamespace,
			Labels: map[string]string{
				constants.LabelIstioInjection: "enabled",
			},
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// AppendOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Append custom images from the subcomponents in the bom
	ctx.Log().Debug("Appending the image overrides for the Prometheus Operator components")
	subcomponents := []string{"prometheus-config-reloader", "alertmanager"}
	kvs, err := appendCustomImageOverrides(ctx, kvs, subcomponents)
	if err != nil {
		return kvs, err
	}

	// Replace default images for subcomponents Alertmanager and Prometheus
	defaultImages := map[string]string{
		// format "subcomponentName": "helmDefaultKey"
		"alertmanager": "prometheusOperator.alertmanagerDefaultBaseImage",
		"prometheus":   "prometheusOperator.prometheusDefaultBaseImage",
	}
	kvs, err = appendDefaultImageOverrides(ctx, kvs, defaultImages)
	if err != nil {
		return kvs, err
	}

	// If the cert-manager component is enabled, use it for webhook certificates, otherwise Prometheus Operator
	// will use the kube-webhook-certgen image
	kvs = append(kvs, bom.KeyValue{
		Key:   "prometheusOperator.admissionWebhooks.certManager.enabled",
		Value: strconv.FormatBool(vzconfig.IsCertManagerEnabled(ctx.EffectiveCR())),
	})

	// If we specify a storage or the prod is used, create a PVC for Prometheus
	resourceRequest, err := common.FindStorageOverride(ctx.EffectiveCR())
	if err != nil {
		return kvs, err
	}
	if ctx.EffectiveCR().Spec.Profile == vzapi.Prod || resourceRequest != nil {
		storage := "50Gi"
		memory := "128Mi"
		if resourceRequest != nil {
			storage = resourceRequest.Storage
			memory = resourceRequest.Memory
		}
		kvs = append(kvs, []bom.KeyValue{
			{
				Key:   "prometheusOperator.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName.resources.requests.storage",
				Value: storage,
			},
			{
				Key:   "prometheusOperator.prometheusSpec.storageSpec.volumeClaimTemplate.spec.storageClassName.resources.requests.memory",
				Value: memory,
			},
		}...)
	}

	// Append the Istio Annotations for prometheus
	kvs = prometheus.AppendIstioAnnotations("prometheus.prometheusSpec.podMetadata.annotations", kvs)

	return kvs, nil
}

// GetHelmOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func GetHelmOverrides(ctx spi.ComponentContext) []vzapi.Overrides {
	if ctx.EffectiveCR().Spec.Components.PrometheusOperator != nil {
		return ctx.EffectiveCR().Spec.Components.PrometheusOperator.ValueOverrides
	}
	return []vzapi.Overrides{}
}

// appendCustomImageOverrides takes a list of subcomponent image names and appends it to the given Helm overrides
func appendCustomImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents []string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	for _, subcomponent := range subcomponents {
		imageOverrides, err := bomFile.BuildImageOverrides(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build the Prometheus Operator image overrides for subcomponent %s: ", subcomponent, err)
		}
		kvs = append(kvs, imageOverrides...)
	}

	return kvs, nil
}

func appendDefaultImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents map[string]string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Prometheus Operator image overrides: ", err)
	}

	for subcomponent, helmKey := range subcomponents {
		images, err := bomFile.GetImageNameList(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to get the image for subcomponent %s from the bom: ", subcomponent, err)
		}
		if len(images) > 0 {
			kvs = append(kvs, bom.KeyValue{Key: helmKey, Value: images[0]})
		}
	}

	return kvs, nil
}

// validatePrometheusOperator checks scenarios in which the Verrazzano CR violates install verification due to Prometheus Operator specifications
func (c prometheusComponent) validatePrometheusOperator(effectiveCR *vzapi.Verrazzano) error {
	// Validate if Prometheus is enabled, Prometheus Operator should be enabled
	prometheus := effectiveCR.Spec.Components.Prometheus
	prometheusEnabled := prometheus != nil && prometheus.Enabled != nil && *prometheus.Enabled
	if !c.IsEnabled(effectiveCR) && prometheusEnabled {
		return fmt.Errorf("Prometheus cannot be enabled if the Prometheus Operator is disabled")
	}

	// Validate Helm value overrides
	if effectiveCR.Spec.Components.PrometheusOperator != nil {
		if err := vzapi.ValidateHelmValueOverrides(effectiveCR.Spec.Components.PrometheusOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}
