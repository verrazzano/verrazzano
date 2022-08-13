// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"fmt"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

const kibanaStatefulSet = "vmi-system-kibana"

// isOSDReady checks if the OpenSearch-Dashboards resources are ready
func isOSDReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	replicas := getReplicaCount(ctx.EffectiveCR())
	if replicas > 0 &&
		!status.StatefulSetsAreReady(ctx.Log(), ctx.Client(), []types.NamespacedName{{
			Name:      kibanaStatefulSet,
			Namespace: ComponentNamespace,
		}}, replicas, prefix) {
		return false
	}

	return common.IsVMISecretReady(ctx)
}

// doesOSDExist is the IsInstalled check
func doesOSDExist(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	statefulSet := []types.NamespacedName{{
		Name:      kibanaStatefulSet,
		Namespace: ComponentNamespace,
	}}
	return status.DoStatefulSetsExist(ctx.Log(), ctx.Client(), statefulSet, 1, prefix)
}

// getReplicaCount - return the OpenSearch-Dashboards replica count
func getReplicaCount(effectiveCR *vzapi.Verrazzano) int32 {
	replicaCount := int32(0)

	if vzconfig.IsKibanaEnabled(effectiveCR) {
		osd := effectiveCR.Spec.Components.Kibana
		if osd == nil || osd.Replicas == nil {
			// Default to one if the component defaulted to being enabled
			replicaCount = 1
		} else {
			replicaCount = *osd.Replicas
		}

	}
	return replicaCount
}
