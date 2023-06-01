// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteragent

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
)

// AppendClusterAgentOverrides Honor the APP_OPERATOR_IMAGE env var if set; this allows an explicit override
// of the verrazzano-application-operator image when set.
func AppendClusterAgentOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Respect the env variable override for the application operator image
	// If it is not set, default to the image given in the BOM
	envImageOverride := os.Getenv(constants.VerrazzanoAppOperatorImageEnvVar)
	if len(envImageOverride) > 0 {
		kvs = append(kvs, bom.KeyValue{
			Key:   "image",
			Value: envImageOverride,
		})
	} else {
		// Get Application Operator image from the BOM
		bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
		if err != nil {
			return nil, err
		}

		images, err := bomFile.BuildImageOverrides("verrazzano-application-operator")
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, images...)
	}
	return kvs, nil
}

// GetOverrides gets the installation overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterAgent != nil {
			return effectiveCR.Spec.Components.ClusterAgent.ValueOverrides
		}
		return []v1alpha1.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterAgent != nil {
			return effectiveCR.Spec.Components.ClusterAgent.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}

	return []v1alpha1.Overrides{}
}
