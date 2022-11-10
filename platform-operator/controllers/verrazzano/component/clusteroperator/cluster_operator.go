// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteroperator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
)

// AppendClusterOperatorOverrides appends the overrides for the Cluster Operator component
func AppendClusterOperatorOverrides(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	return kvs, nil
}

// isClusterOperatorReady checks if the cluster operator deployment is ready
func (c clusterOperatorComponent) isClusterOperatorReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DeploymentNames, 1, prefix)
}

// GetOverrides gets the install overrides for the Cluster Operator component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterOperator != nil {
			return effectiveCR.Spec.Components.ClusterOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ClusterOperator != nil {
			return effectiveCR.Spec.Components.ClusterOperator.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}
