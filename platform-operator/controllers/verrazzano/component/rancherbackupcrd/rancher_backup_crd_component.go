// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherbackupcrd

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	// ComponentName is the name of the component
	ComponentName = "rancher-backup-crd"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.RancherBackupNamesSpace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "rancher-backup-crd"
	// ChartDir is the name of the directory for third party helm charts
	ChartDir = "rancher-backup-crd"
)

var (
	componentPrefix   = fmt.Sprintf("Component %s", ComponentName)
	rancherDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	deployments = []types.NamespacedName{
		rancherDeployment,
	}
)

type rancherBackupCrdHelmComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return helm.HelmComponent{
		ReleaseName:               ComponentName,
		JSONName:                  ComponentJSONName,
		ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ChartDir),
		ChartNamespace:            ComponentNamespace,
		IgnoreNamespaceOverride:   true,
		SupportsOperatorInstall:   true,
		SupportsOperatorUninstall: true,
		MinVerrazzanoVersion:      constants.VerrazzanoVersion1_4_0,
		//ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "rancher-backup-override-static-values.yaml"),
		//Dependencies:              []string{rancher.ComponentName},
		Dependencies: []string{},
	}
}

// IsEnabled returns true only if Rancher Backup component is explicitly enabled
// in the Verrazzano CR.
func (rb rancherBackupCrdHelmComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	return true
}

// IsInstalled returns true only if Rancher Backup is installed on the system
//func (rb rancherBackupHelmComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
//	for _, nsn := range deployments {
//		if err := ctx.Client().Get(context.TODO(), nsn, &appsv1.Deployment{}); err != nil {
//			if errors.IsNotFound(err) {
//				return false, nil
//			}
//			// Unexpected error
//			return false, err
//		}
//	}
//	return true, nil
//}

// validateRancherBackup checks scenarios in which the Verrazzano CR violates install verification
//func (rb rancherBackupHelmComponent) validateRancherBackup(vz *vzapi.Verrazzano) error {
//	// Validate install overrides
//	if vz.Spec.Components.RancherBackup != nil {
//		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.RancherBackup.ValueOverrides); err != nil {
//			return err
//		}
//	}
//	return nil
//}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
//func (rb rancherBackupHelmComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
//	if ctx.EffectiveCR().Spec.Components.RancherBackup == nil {
//		return false
//	}
//	if ctx.EffectiveCR().Spec.Components.RancherBackup.MonitorChanges != nil {
//		return *ctx.EffectiveCR().Spec.Components.RancherBackup.MonitorChanges
//	}
//	return true
//}

func (rb rancherBackupCrdHelmComponent) PreInstall(ctx spi.ComponentContext) error {
	return ensureRancherBackupNamespace(ctx)
}

// IsReady checks if the Velero objects are ready
//func (rb rancherBackupHelmComponent) IsReady(ctx spi.ComponentContext) bool {
//	return isRancherBackupOperatorReady(ctx)
//}
//
//func (rb rancherBackupHelmComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
//	return nil
//}
//
//// ValidateUpgrade verifies the upgrade of the Verrazzano object
//func (rb rancherBackupHelmComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
//	if rb.IsEnabled(old) && !rb.IsEnabled(new) {
//		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
//	}
//	return rb.validateRancherBackup(new)
//}
//
//// PostUninstall processing for RancherBackup
//func (rb rancherBackupHelmComponent) PostUninstall(context spi.ComponentContext) error {
//	res := resource.Resource{
//		Name:   ComponentNamespace,
//		Client: context.Client(),
//		Object: &corev1.Namespace{},
//		Log:    context.Log(),
//	}
//	// Remove finalizers from the cattle-resources-system namespace to avoid hanging namespace deletion
//	// and delete the namespace
//	return res.RemoveFinalizersAndDelete()
//}

func ensureRancherBackupNamespace(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Creating namespace %s for Rancher Backup.", ComponentNamespace)
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}
