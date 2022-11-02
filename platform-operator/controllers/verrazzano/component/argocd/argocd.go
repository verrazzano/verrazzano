// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Constants for Kubernetes resource names
const (
	defaultSecretNamespace = "cert-manager"
	defaultVerrazzanoName  = "verrazzano-ca-certificate-secret"
)

// GetOverrides returns the install overrides from v1beta1.Verrazzano CR
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.ArgoCD != nil {
			return effectiveCR.Spec.Components.ArgoCD.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.ArgoCD != nil {
			return effectiveCR.Spec.Components.ArgoCD.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// isArgoCDReady checks the state of the expected argocd deployments and returns true if they are in a ready state
func isArgoCDReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      common.ArgoCDApplicationSetController,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDDexServer,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDNotificationController,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDRedis,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDRepoServer,
			Namespace: ComponentNamespace,
		},
		{
			Name:      common.ArgoCDServer,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}
