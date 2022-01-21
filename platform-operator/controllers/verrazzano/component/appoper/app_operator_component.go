// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package appoper

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"path/filepath"
	"strings"

	vmcv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

type applicationOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return applicationOperatorComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendApplicationOperatorOverrides,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			ReadyStatusFunc:         IsApplicationOperatorReady,
			Dependencies:            []string{oam.ComponentName, istio.ComponentName},
			PreUpgradeFunc:          ApplyCRDYaml,
		},
	}
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
				errorList = append(errorList, fmt.Sprintf("failed to delete ClusterRoleBinding %s, error: %s", vmc.Name, err.Error()))
			} else {
				ctx.Log().Debugf("Deleted ClusterRoleBinding %s", clusterRoleBinding.Name)
			}
		}
	}
	if len(errorList) > 0 {
		return errors.New(strings.Join(errorList, "\n"))
	}
	return nil

}

// IsEnabled applicationOperator-specific enabled check for installation
func (c applicationOperatorComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.ApplicationOperator
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
