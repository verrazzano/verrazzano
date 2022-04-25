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
	ComponentName = "jaeger-operator"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoMonitoringNamespace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "jaegerOperator"
)

var (
	componentPrefix          = fmt.Sprintf("Component %s", ComponentName)
	jaegerOperatorDeployment = types.NamespacedName{
		Name:      ComponentName,
		Namespace: ComponentNamespace,
	}
	deployments = []types.NamespacedName{
		jaegerOperatorDeployment,
	}
)

//jaegerOperatorComponent is stubbed if any properties need to be added in the future
type jaegerOperatorComponent struct{}

func NewComponent() spi.Component {
	return jaegerOperatorComponent{}
}

func (c jaegerOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return ensureVerrazzanoMonitoringNamespace(ctx)
}

func (c jaegerOperatorComponent) Upgrade(ctx spi.ComponentContext) error {
	return componentInstall(ctx)
}

func (c jaegerOperatorComponent) Install(ctx spi.ComponentContext) error {
	return componentInstall(ctx)
}

func (c jaegerOperatorComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

func (c jaegerOperatorComponent) IsReady(context spi.ComponentContext) bool {
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, componentPrefix)
}

// IsEnabled returns true only if the Jaeger Operator is explicitly enabled
// in the Verrazzano CR.
func (c jaegerOperatorComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.JaegerOperator
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

func (c jaegerOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
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

func (c jaegerOperatorComponent) Name() string {
	return ComponentName
}

func (c jaegerOperatorComponent) GetDependencies() []string {
	return []string{}
}

func (c jaegerOperatorComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_3_0
}

func (c jaegerOperatorComponent) GetJSONName() string {
	return ComponentJSONName
}

func (c jaegerOperatorComponent) IsOperatorInstallSupported() bool {
	return true
}

// ##### Only interface stubs below #####

func (c jaegerOperatorComponent) PostInstall(_ spi.ComponentContext) error {
	return nil
}

func (c jaegerOperatorComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c jaegerOperatorComponent) PostUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c jaegerOperatorComponent) GetIngressNames(_ spi.ComponentContext) []types.NamespacedName {
	return nil
}

func (c jaegerOperatorComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return nil
}

func (c jaegerOperatorComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

func (c jaegerOperatorComponent) ValidateUpdate(_, _ *vzapi.Verrazzano) error {
	return nil
}
