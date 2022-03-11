// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
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

func ApplyCRDYaml(log vzlog.VerrazzanoLogger, c client.Client, _ string, _ string, _ string) error {
	path := filepath.Join(config.GetHelmAppOpChartsDir(), "/crds")
	yamlApplier := k8sutil.NewYAMLApplier(c, "")
	return yamlApplier.ApplyD(path)
}
