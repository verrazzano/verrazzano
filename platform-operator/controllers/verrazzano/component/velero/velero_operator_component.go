// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ComponentName is the name of the component
	ComponentName = "velero"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VeleroNameSpace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "velero"

	resticConfigmapFile = "/velero/velero-configmap.yaml"
)

var (
	componentPrefix  = fmt.Sprintf("Component %s", ComponentName)
	veleroDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	deployments = []types.NamespacedName{
		veleroDeployment,
	}
)

//veleroComponent is stubbed if any properties need to be added in the future
type veleroComponent struct{}

func NewComponent() spi.Component {
	return veleroComponent{}
}

func (v veleroComponent) PreInstall(ctx spi.ComponentContext) error {
	return nil
}

func (v veleroComponent) Upgrade(ctx spi.ComponentContext) error {
	return componentInstall(ctx)
}

func (v veleroComponent) Install(ctx spi.ComponentContext) error {
	ctx.Log().Infof("+++ Install Velero Operator Triggered +++")
	return componentInstall(ctx)
}

func (v veleroComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

func (v veleroComponent) IsReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}

// IsEnabled returns true only if the Jaeger Operator is explicitly enabled
// in the Verrazzano CR.
func (v veleroComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Velero
	fmt.Printf(" COMP = %v", comp)
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

func (v veleroComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	for _, nsn := range deployments {
		if err := ctx.Client().Get(context.TODO(), nsn, &appsv1.Deployment{}); err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			// Unexpected error
			return false, err
		}
	}
	return true, nil
}

func (v veleroComponent) Name() string {
	return ComponentName
}

func (v veleroComponent) GetDependencies() []string {
	return []string{}
}

func (v veleroComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_3_0
}

func (v veleroComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetHelmOverrides returns the Helm override sources for a component
func (v veleroComponent) GetOverrides(_ *vzapi.Verrazzano) []vzapi.Overrides {
	return []vzapi.Overrides{}
}

// MonitorOverrides indicates whether monitoring of Helm override sources is enabled for a component
func (v veleroComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

func (v veleroComponent) IsOperatorInstallSupported() bool {
	return true
}

// ##### Only interface stubs below #####

func (v veleroComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (v veleroComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (v veleroComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (v veleroComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return nil
}

func (v veleroComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return nil
}

func (v veleroComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

func (v veleroComponent) ValidateUpdate(_, _ *vzapi.Verrazzano) error {
	return nil
}
