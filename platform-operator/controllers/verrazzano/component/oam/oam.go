// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	pvcClusterRoleName         = "oam-kubernetes-runtime-pvc"
	istioClusterRoleName       = "oam-kubernetes-runtime-istio"
	certClusterRoleName        = "oam-kubernetes-runtime-certificate"
	aggregateToControllerLabel = "rbac.oam.dev/aggregate-to-controller"
)

// isOAMReady checks if the OAM operator deployment is ready
func isOAMReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// ensureClusterRoles creates or updates additional OAM cluster roles during install and upgrade
func ensureClusterRoles(ctx spi.ComponentContext) error {
	// add a cluster role that allows the OAM operator to manage persistent volume claim workloads
	pvcClusterRole := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: pvcClusterRoleName}}

	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &pvcClusterRole, func() error {
		if pvcClusterRole.Labels == nil {
			pvcClusterRole.Labels = make(map[string]string)
		}
		// this label triggers cluster role aggregation into the oam-kubernetes-runtime cluster role
		pvcClusterRole.Labels[aggregateToControllerLabel] = "true"
		pvcClusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{corev1.ResourcePersistentVolumeClaims.String()},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"deletecollection",
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	// add a cluster role that allows the OAM operator to manage istio resources
	istioClusterRole := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: istioClusterRoleName}}

	_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &istioClusterRole, func() error {
		if istioClusterRole.Labels == nil {
			istioClusterRole.Labels = make(map[string]string)
		}
		// this label triggers cluster role aggregation into the oam-kubernetes-runtime cluster role
		istioClusterRole.Labels[aggregateToControllerLabel] = "true"
		istioClusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"networking.istio.io", "install.istio.io", "security.istio.io", "telemetry.istio.io"},
				Resources: []string{"*"},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
					"deletecollection",
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	// add a cluster role that allows the OAM operator to manage secret resources
	certClusterRole := rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: certClusterRoleName}}

	_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &certClusterRole, func() error {
		if certClusterRole.Labels == nil {
			certClusterRole.Labels = make(map[string]string)
		}
		// this label triggers cluster role aggregation into the oam-kubernetes-runtime cluster role
		certClusterRole.Labels[aggregateToControllerLabel] = "true"
		certClusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{"cert-manager.io"},
				Resources: []string{"*"},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
					"deletecollection",
				},
			},
		}
		return nil
	})

	return err
}

func deleteOAMClusterRoles(client client.Client, log vzlog.VerrazzanoLogger) error {
	ctx := context.TODO()
	clusterRoles := []*rbacv1.ClusterRole{
		{ObjectMeta: metav1.ObjectMeta{Name: pvcClusterRoleName}},
		{ObjectMeta: metav1.ObjectMeta{Name: istioClusterRoleName}},
		{ObjectMeta: metav1.ObjectMeta{Name: certClusterRoleName}},
	}
	for _, role := range clusterRoles {
		log.Progressf("Deleting OAM clusterrole %s", role.Name)
		if err := client.Delete(ctx, role); err != nil {
			return err
		}
	}
	return nil
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.OAM != nil {
			return effectiveCR.Spec.Components.OAM.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.OAM != nil {
			return effectiveCR.Spec.Components.OAM.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
