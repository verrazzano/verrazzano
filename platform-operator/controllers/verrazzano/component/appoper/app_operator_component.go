// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"strings"

	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-application-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "applicationOperator"

type applicationOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return applicationOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			AppendOverridesFunc:       AppendApplicationOperatorOverrides,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0]",
			Dependencies:              []string{networkpolicies.ComponentName, oam.ComponentName, istio.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      ComponentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsReady component check
func (c applicationOperatorComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return c.isApplicationOperatorReady(context)
	}
	return false
}

// PreUpgrade processing for the application-operator
func (c applicationOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	err := common.ApplyCRDYaml(ctx, config.GetHelmAppOpChartsDir())
	if err != nil {
		return err
	}
	err = labelAnnotateTraitDefinitions(ctx.Client())
	if err != nil {
		return err
	}
	return labelAnnotateWorkloadDefinitions(ctx.Client())
}

// PostUpgrade processing for the application-operator
func (c applicationOperatorComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("application-operator post-upgrade")

	var clientCtx = context.TODO()

	// In v1.1 the use of ClusterRoleBindings to control access for a managed cluster
	// was changed to use RoleBindings instead.  Delete any ClusterRoleBindings left on
	// the system for multicluster.
	vmcList := vmcv1alpha1.VerrazzanoManagedClusterList{}
	err := ctx.Client().List(clientCtx, &vmcList)
	if err != nil {
		return err
	}
	var errorList []string
	for _, vmc := range vmcList.Items {
		clusterRoleBinding := rbacv1.ClusterRoleBinding{}
		err := ctx.Client().Get(clientCtx, types.NamespacedName{Name: fmt.Sprintf("verrazzano-cluster-%s", vmc.Name)}, &clusterRoleBinding)
		if err == nil {
			// Delete the ClusterRoleBinding
			err = ctx.Client().Delete(clientCtx, &clusterRoleBinding)
			if err != nil {
				errorList = append(errorList, fmt.Sprintf("Failed to delete ClusterRoleBinding %s, error: %v", vmc.Name, err.Error()))
			} else {
				ctx.Log().Debugf("Deleted ClusterRoleBinding %s", clusterRoleBinding.Name)
			}
		}
	}
	if len(errorList) > 0 {
		return ctx.Log().ErrorfNewErr(strings.Join(errorList, ";"))
	}
	return nil

}

// IsEnabled applicationOperator-specific enabled check for installation
func (c applicationOperatorComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsApplicationOperatorEnabled(effectiveCR)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c applicationOperatorComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c applicationOperatorComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c applicationOperatorComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.ApplicationOperator != nil {
		if ctx.EffectiveCR().Spec.Components.ApplicationOperator.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.ApplicationOperator.MonitorChanges
		}
		return true
	}
	return false
}
