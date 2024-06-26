// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
)

const (
	kibanaDeployment = "vmi-system-osd"
	osdDeployment    = "opensearch-dashboards"
)

func getOSDDeployments(ctx spi.ComponentContext) []types.NamespacedName {
	isLegacyOSD, err := common.IsLegacyOSD(ctx)
	if err != nil {
		ctx.Log().ErrorfThrottled("Failed to get VMI, considering legacy OSD to be disabled: %v", err)
	}
	if isLegacyOSD {
		return []types.NamespacedName{
			{
				Name:      kibanaDeployment,
				Namespace: ComponentNamespace,
			},
		}
	}
	return []types.NamespacedName{
		{
			Name:      osdDeployment,
			Namespace: constants.VerrazzanoLoggingNamespace,
		},
	}
}

// isOSDReady checks if the OpenSearch-Dashboards resources are ready
func isOSDReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if vzcr.IsOpenSearchDashboardsEnabled(ctx.EffectiveCR()) {
		replicas := int32(1)
		if ctx.EffectiveCR().Spec.Components.Kibana != nil && ctx.EffectiveCR().Spec.Components.Kibana.Replicas != nil {
			replicas = *ctx.EffectiveCR().Spec.Components.Kibana.Replicas
		}
		if !ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), getOSDDeployments(ctx), replicas, prefix) {
			return false
		}
	}
	return common.IsVMISecretReady(ctx)
}

// doesOSDExist is the IsInstalled check
func doesOSDExist(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	deploy := []types.NamespacedName{{
		Name:      kibanaDeployment,
		Namespace: ComponentNamespace,
	}}
	return ready.DoDeploymentsExist(ctx.Log(), ctx.Client(), deploy, 1, prefix)
}
