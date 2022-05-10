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
	ComponentName = "velero-operator"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VeleroNameSpace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "veleroOperator"
)

var (
	componentPrefix          = fmt.Sprintf("Component %s", ComponentName)
	veleroOperatorDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	deployments = []types.NamespacedName{
		veleroOperatorDeployment,
	}
)

//veleroOperatorComponent is stubbed if any properties need to be added in the future
type veleroOperatorComponent struct{}

func NewComponent() spi.Component {
	return veleroOperatorComponent{}
}

func (c veleroOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return ensureVerrazzanoMonitoringNamespace(ctx)
}

func (c veleroOperatorComponent) Upgrade(ctx spi.ComponentContext) error {
	return componentInstall(ctx)
}

func (c veleroOperatorComponent) Install(ctx spi.ComponentContext) error {
	return componentInstall(ctx)
}

func (c veleroOperatorComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

func (c veleroOperatorComponent) IsReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}

// IsEnabled returns true only if the Jaeger Operator is explicitly enabled
// in the Verrazzano CR.
func (c veleroOperatorComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.VeleroOperator
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

func (c veleroOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
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

func (c veleroOperatorComponent) Name() string {
	return ComponentName
}

func (c veleroOperatorComponent) GetDependencies() []string {
	return []string{}
}

func (c veleroOperatorComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_3_0
}

func (c veleroOperatorComponent) GetJSONName() string {
	return ComponentJSONName
}

func (c veleroOperatorComponent) IsOperatorInstallSupported() bool {
	return true
}

// ##### Only interface stubs below #####

func (c veleroOperatorComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (c veleroOperatorComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c veleroOperatorComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c veleroOperatorComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return nil
}

func (c veleroOperatorComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return nil
}

func (c veleroOperatorComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

func (c veleroOperatorComponent) ValidateUpdate(_, _ *vzapi.Verrazzano) error {
	return nil
}
