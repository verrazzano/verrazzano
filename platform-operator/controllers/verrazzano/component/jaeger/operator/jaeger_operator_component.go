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
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
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

// GetOverrides returns the Helm override sources for a component
func (c jaegerOperatorComponent) GetOverrides(_ *vzapi.Verrazzano) []vzapi.Overrides {
	return []vzapi.Overrides{}
}

// MonitorOverrides indicates whether monitoring of Helm override sources is enabled for a component
func (c jaegerOperatorComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

func (c jaegerOperatorComponent) IsOperatorInstallSupported() bool {
	return true
}

// ##### Only interface stubs below #####

func (c jaegerOperatorComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Jaeger post-install")
	return c.createOrUpdateJaegerResources(ctx)
}

func (c jaegerOperatorComponent) PreUpgrade(_ spi.ComponentContext) error {
	return nil
}

func (c jaegerOperatorComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Jaeger post-upgrade")
	return c.createOrUpdateJaegerResources(ctx)
}

func (c jaegerOperatorComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName
	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		ingressNames = []types.NamespacedName{
			{
				Namespace: ComponentNamespace,
				Name:      constants.JaegerIngress,
			},
		}
	}
	return ingressNames
}

func (c jaegerOperatorComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		certificateNames = append(certificateNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      jaegerCertificateName,
		})
	}
	return certificateNames
}

func (c jaegerOperatorComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

func (c jaegerOperatorComponent) ValidateUpdate(_, _ *vzapi.Verrazzano) error {
	return nil
}

// createOrUpdateKialiResources create or update related Kiali resources
func (c jaegerOperatorComponent) createOrUpdateJaegerResources(ctx spi.ComponentContext) error {
	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) {
		if err := createOrUpdateJaegerIngress(ctx, ComponentNamespace); err != nil {
			return err
		}
	}
	return nil
}
