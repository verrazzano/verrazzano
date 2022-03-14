// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	return kvs, nil
}

// isApplicationOperatorReady checks if the application operator deployment is ready
func isApplicationOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []status.PodReadyCheck{
		{
			NamespacedName: types.NamespacedName{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
			},
			LabelSelector: labels.Set{"app": ComponentName}.AsSelector(),
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

func applyCRDYaml(c client.Client) error {
	path := filepath.Join(config.GetHelmAppOpChartsDir(), "/crds")
	yamlApplier := k8sutil.NewYAMLApplier(c, "")
	return yamlApplier.ApplyD(path)
}

// Add labels/annotations required by Helm to the Verrazzano installed trait definitions.  Originally, the
// trait definitions were included in the helm charts crds directory, and they did not get installed with the required
// labels/annotations.  Adding the labels/annotations allows helm upgrade to proceed without errors.
func labelTraitDefinitions(ctx spi.ComponentContext) error {
	traitDefinitions := []string{
		"ingresstraits.oam.verrazzano.io",
		"loggingtraits.oam.verrazzano.io",
		"metricstraits.oam.verrazzano.io",
	}

	for _, traitDefinition := range traitDefinitions {
		trait := oamv1alpha2.TraitDefinition{}
		err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: traitDefinition}, &trait)
		if err != nil {
			return err
		}
		// Add label required by Helm
		if trait.Labels == nil {
			trait.Labels = map[string]string{}
		}
		trait.Labels["app.kubernetes.io/managed-by"] = "Helm"
		// Add annotations required by Helm
		if trait.Annotations == nil {
			trait.Annotations = map[string]string{}
		}
		trait.Annotations["meta.helm.sh/release-name"] = ComponentName
		trait.Annotations["meta.helm.sh/release-namespace"] = ComponentNamespace

		err = ctx.Client().Update(context.TODO(), &trait)
		if err != nil {
			return err
		}
	}
	return nil
}

// Add labels/annotations required by Helm to the Verrazzano installed workload definitions.  Originally, the
// workload definitions were included in the helm charts crds directory, and they did not get installed with the required
// labels/annotations.  Adding the labels/annotations allows helm upgrade to proceed without errors.
func labelWorkloadDefinitions(ctx spi.ComponentContext) error {
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
		err := ctx.Client().Get(context.TODO(), types.NamespacedName{Name: workloadDefinition}, &workload)
		if err != nil {
			return err
		}
		// Add label required by Helm
		if workload.Labels == nil {
			workload.Labels = map[string]string{}
		}
		workload.Labels["app.kubernetes.io/managed-by"] = "Helm"
		// Add annotations required by Helm
		if workload.Annotations == nil {
			workload.Annotations = map[string]string{}
		}
		workload.Annotations["meta.helm.sh/release-name"] = ComponentName
		workload.Annotations["meta.helm.sh/release-namespace"] = ComponentNamespace

		err = ctx.Client().Update(context.TODO(), &workload)
		if err != nil {
			return err
		}
	}
	return nil
}
