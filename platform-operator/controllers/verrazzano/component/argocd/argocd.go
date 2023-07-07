// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Constants for Kubernetes resource names
const (
	defaultSecretNamespace = "cert-manager"
)

// GetOverrides returns the install overrides from either v1alpha1 or v1beta1.Verrazzano CR
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

// remove stale argo app, app set, and project resources
func removeArgoResources(ctx spi.ComponentContext) error {
	if err := removeArgoResourceKind(ctx, common.ArgoCDKindApplication); err != nil {
		return err
	}
	if err := removeArgoResourceKind(ctx, common.ArgoCDKindApplicationSet); err != nil {
		return err
	}
	if err := removeArgoResourceKind(ctx, common.ArgoCDKindAppProject); err != nil {
		return err
	}
	return nil
}

// removeArgoResourceKind iterates through the ArgoCD resources installed, removes the finalizer, and deletes them
func removeArgoResourceKind(ctx spi.ComponentContext, kind string) error {
	c := ctx.Client()
	resources := unstructured.UnstructuredList{}
	resources.SetGroupVersionKind(common.GetArgoProjAPIGVRForResource(kind))
	if err := c.List(context.TODO(), &resources, client.InNamespace(constants.ArgoCDNamespace)); err != nil {
		return err
	}
	for _, resource := range resources.Items {
		// for each resource delete the finalizer and then delete the resource
		_, err := controllerruntime.CreateOrUpdate(context.TODO(), c, &resource, func() error {
			resource.SetFinalizers([]string{})
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing finalizer stale ArgoCD %s resource %s", kind, resource.GetName())
		}

		err = c.Delete(context.TODO(), &resource)
		if err != nil {
			return fmt.Errorf("Error removing finalizer stale ArgoCD %s resource %s", kind, resource.GetName())

		}
	}
	return nil
}
