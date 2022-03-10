// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appoper

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"path/filepath"
	"strings"

	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
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

type applicationOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return applicationOperatorComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendApplicationOperatorOverrides,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			Dependencies:            []string{oam.ComponentName, istio.ComponentName},
			PreUpgradeFunc:          ApplyCRDYaml,
		},
	}
}

// IsReady component check
func (c applicationOperatorComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isApplicationOperatorReady(context)
	}
	return false
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
func (c applicationOperatorComponent) IsEnabled(ctx spi.ComponentContext) bool {
	return isApplicationOperatorComponentEnabled(ctx.EffectiveCR())
}

func isApplicationOperatorComponentEnabled(vz *vzapi.Verrazzano) bool {
	comp := vz.Spec.Components.ApplicationOperator
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c applicationOperatorComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c applicationOperatorComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if isApplicationOperatorComponentEnabled(old) && !isApplicationOperatorComponentEnabled(new) {
		return fmt.Errorf("can not disable applicationOperator")
	}
	return nil
}
