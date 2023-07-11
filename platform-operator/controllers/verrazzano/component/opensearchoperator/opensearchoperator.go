// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

const (
	opensearchOperatorDeploymentName = "opensearch-operator-controller-manager"

	opsterOSDIngressName = "opster-osd"

	opsterOSIngressName = "opster-os"

	opsterOSService = "opensearch"

	opsterOSDService = "opensearch-dashboards"

	securityconfigSecretName = "securityconfig-secret"
)

func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), getDeploymentList(), 1, getPrefix(ctx))
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Keycloak != nil {
			return effectiveCR.Spec.Components.Keycloak.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

func GetMergedNodes(ctx spi.ComponentContext) {
	//legacyNodes := getNodePoolFromNodes(ctx.EffectiveCR().Spec.Components.Elasticsearch.Nodes)
	//currentNodes := getNodesFromOverrides(ctx.EffectiveCR().Spec.Components.OpenSearchOperator)
}

func getDeploymentList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      opensearchOperatorDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
}

func getIngressList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      opsterOSIngressName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      opsterOSDIngressName,
			Namespace: ComponentNamespace,
		},
	}
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}
