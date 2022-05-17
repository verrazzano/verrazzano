// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"strconv"

	vmoconst "github.com/verrazzano/verrazzano-monitoring-operator/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	deploymentName  = "prometheus-operator-kube-p-operator"
	istioVolumeName = "istio-certs-dir"
)

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
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), prometheus.GetVerrazzanoMonitoringNamespace(), func() error {
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
	if resourceRequest != nil {
		storage := resourceRequest.Storage
		memory := resourceRequest.Memory
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

	// Append the Istio Annotations for Prometheus
	kvs, err = appendIstioOverrides(ctx,
		"prometheus.prometheusSpec.podMetadata.annotations",
		"prometheus.prometheusSpec.volumeMounts",
		"prometheus.prometheusSpec.volumes",
		kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed applying the Istio Overrides for Prometheus")
	}
	return kvs, nil
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
	if !c.IsEnabled(effectiveCR) && vzconfig.IsPrometheusEnabled(effectiveCR) {
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

// appendIstioOverrides appends Istio annotations necessary for Prometheus in Istio
// Istio is required on the Prometheus for mTLS between it and Verrazzano applications
func appendIstioOverrides(ctx spi.ComponentContext, annotationsKey, volumeMountKey, volumeKey string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Set the Istio annotation on Prometheus to exclude Keycloak HTTP Service IP address.
	// The includeOutboundIPRanges implies all others are excluded.
	// This is done by adding the traffic.sidecar.istio.io/includeOutboundIPRanges=<Keycloak IP>/32 annotation.
	svc := corev1.Service{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: "keycloak-http", Namespace: constants.KeycloakNamespace}, &svc)
	if err != nil {
		if !errors.IsNotFound(err) {
			return kvs, ctx.Log().ErrorfNewErr("Failed to get keycloak-http service: %v", err)
		}
	}
	outboundIP := fmt.Sprintf("%s/32", svc.Spec.ClusterIP)
	if svc.Spec.ClusterIP == "" {
		outboundIP = "0.0.0.0/0"
	}

	// Istio annotations that will copy the volume mount for the Istio certs to the envoy sidecar
	// The last annotation allows envoy to intercept only requests from the Keycloak Service IP
	annotations := map[string]string{
		`proxy\.istio\.io/config`:                             fmt.Sprintf(`{"proxyMetadata":{ "OUTPUT_CERTS": "%s"}}`, vmoconst.IstioCertsMountPath),
		`sidecar\.istio\.io/userVolumeMount`:                  fmt.Sprintf(`[{"name": "istio-certs-dir", "mountPath": "%s"}]`, vmoconst.IstioCertsMountPath),
		`traffic\.sidecar\.istio\.io/includeOutboundIPRanges`: outboundIP,
	}
	for key, value := range annotations {
		kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s.%s", annotationsKey, key), Value: value})
	}

	// Volume mount on the Prometheus container to mount the Istio-generated certificates
	vm := corev1.VolumeMount{
		Name:      istioVolumeName,
		MountPath: vmoconst.IstioCertsMountPath,
	}
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].name", volumeMountKey), Value: vm.Name})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].mountPath", volumeMountKey), Value: vm.MountPath})

	// Volume annotation to enable an in-memory location for Istio to place and serve certificates
	vol := corev1.Volume{
		Name: istioVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].name", volumeKey), Value: vol.Name})
	kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("%s[0].emptyDir.medium", volumeKey), Value: string(vol.VolumeSource.EmptyDir.Medium)})

	return kvs, nil
}

// GetOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func GetOverrides(ctx spi.ComponentContext) []vzapi.Overrides {
	comp := ctx.EffectiveCR().Spec.Components.PrometheusOperator
	if comp == nil {
		return []vzapi.Overrides{}
	}
	if ctx.EffectiveCR().Spec.Components.PrometheusOperator != nil {
		return ctx.EffectiveCR().Spec.Components.PrometheusOperator.ValueOverrides
	}
	return []vzapi.Overrides{}
}
