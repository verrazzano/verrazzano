// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	helmManagedByLabel             = "app.kubernetes.io/managed-by"
	helmReleaseNameAnnotation      = "meta.helm.sh/release-name"
	helmReleaseNamespaceAnnotation = "meta.helm.sh/release-namespace"
)

// AppendApplicationOperatorOverrides Honor the APP_OPERATOR_IMAGE env var if set; this allows an explicit override
// of the verrazzano-application-operator image when set.
func AppendApplicationOperatorOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	envImageOverride := os.Getenv(constants.VerrazzanoAppOperatorImageEnvVar)
	if len(envImageOverride) > 0 {
		kvs = append(kvs, bom.KeyValue{
			Key:   "image",
			Value: envImageOverride,
		})
	}

	// Create a Bom and get the Key Value overrides
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}

	// Get fluentd and istio proxy images
	var fluentdImage string
	var istioProxyImage string
	images, err := bomFile.BuildImageOverrides("verrazzano")
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		if image.Key == "logging.fluentdImage" {
			fluentdImage = image.Value
		}
		if image.Key == "monitoringOperator.istioProxyImage" {
			istioProxyImage = image.Value
		}
	}
	if len(fluentdImage) == 0 {
		return nil, compContext.Log().ErrorNewErr("Failed to find logging.fluentdImage in BOM")
	}
	if len(istioProxyImage) == 0 {
		return nil, compContext.Log().ErrorNewErr("Failed to find monitoringOperator.istioProxyImage in BOM")
	}

	// fluentdImage for ENV DEFAULT_FLUENTD_IMAGE
	kvs = append(kvs, bom.KeyValue{
		Key:   "fluentdImage",
		Value: fluentdImage,
	})

	// istioProxyImage for ENV ISTIO_PROXY_IMAGE
	kvs = append(kvs, bom.KeyValue{
		Key:   "istioProxyImage",
		Value: istioProxyImage,
	})

	// get weblogicMonitoringExporter image
	var weblogicMonitoringExporterImage string
	images, err = bomFile.BuildImageOverrides("weblogic-operator")
	if err != nil {
		return nil, err
	}
	for _, image := range images {
		if image.Key == "weblogicMonitoringExporterImage" {
			weblogicMonitoringExporterImage = image.Value
		}
	}
	if len(weblogicMonitoringExporterImage) == 0 {
		return nil, compContext.Log().ErrorNewErr("Failed to find weblogicMonitoringExporterImage in BOM")
	}

	// weblogicMonitoringExporterImage for ENV WEBLOGIC_MONITORING_EXPORTER_IMAGE
	kvs = append(kvs, bom.KeyValue{
		Key:   "weblogicMonitoringExporterImage",
		Value: weblogicMonitoringExporterImage,
	})

	return kvs, nil
}

// isApplicationOperatorReady checks if the application operator deployment is ready
func isApplicationOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// Add label/annotations required by Helm to the Verrazzano installed trait definitions.  Originally, the
// trait definitions were included in the helm charts crds directory, and they did not get installed with the required
// label/annotations.  Adding the label/annotations allows helm upgrade to proceed without errors.
func labelAnnotateTraitDefinitions(c client.Client) error {
	traitDefinitions := []string{
		"ingresstraits.oam.verrazzano.io",
		"loggingtraits.oam.verrazzano.io",
		"metricstraits.oam.verrazzano.io",
	}

	for _, traitDefinition := range traitDefinitions {
		trait := oamv1alpha2.TraitDefinition{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: traitDefinition}, &trait)
		// loggingtraits was not installed in earlier versions of Verrazzano so just
		// continue on to next trait definition in that case.
		if errors.IsNotFound(err) && traitDefinition == "loggingtraits.oam.verrazzano.io" {
			continue
		}
		if err != nil {
			return err
		}
		// Add label required by Helm
		if trait.Labels == nil {
			trait.Labels = map[string]string{}
		}
		trait.Labels[helmManagedByLabel] = "Helm"
		// Add annotations required by Helm
		if trait.Annotations == nil {
			trait.Annotations = map[string]string{}
		}
		trait.Annotations[helmReleaseNameAnnotation] = ComponentName
		trait.Annotations[helmReleaseNamespaceAnnotation] = ComponentNamespace

		err = c.Update(context.TODO(), &trait)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add label/annotations required by Helm to the Verrazzano installed workload definitions.  Originally, the
// workload definitions were included in the helm charts crds directory, and they did not get installed with the required
// label/annotations.  Adding the label/annotations allows helm upgrade to proceed without errors.
func labelAnnotateWorkloadDefinitions(c client.Client) error {
	workloadDefinitions := []string{
		"coherences.coherence.oracle.com",
		"deployments.apps",
		"domains.weblogic.oracle",
		"verrazzanocoherenceworkloads.oam.verrazzano.io",
		"verrazzanohelidonworkloads.oam.verrazzano.io",
		"verrazzanoweblogicworkloads.oam.verrazzano.io",
	}

	for _, workloadDefinition := range workloadDefinitions {
		workload := oamv1alpha2.WorkloadDefinition{}
		err := c.Get(context.TODO(), types.NamespacedName{Name: workloadDefinition}, &workload)
		if err != nil {
			return err
		}
		// Add label required by Helm
		if workload.Labels == nil {
			workload.Labels = map[string]string{}
		}
		workload.Labels[helmManagedByLabel] = "Helm"
		// Add annotations required by Helm
		if workload.Annotations == nil {
			workload.Annotations = map[string]string{}
		}
		workload.Annotations[helmReleaseNameAnnotation] = ComponentName
		workload.Annotations[helmReleaseNamespaceAnnotation] = ComponentNamespace

		err = c.Update(context.TODO(), &workload)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.ApplicationOperator != nil {
			return effectiveCR.Spec.Components.ApplicationOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ApplicationOperator != nil {
			return effectiveCR.Spec.Components.ApplicationOperator.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
