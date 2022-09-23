// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"path/filepath"

	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"k8s.io/apimachinery/pkg/runtime"

	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/opensearch"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
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
	// ChartDir is the relative directory path for Jaeger Operator chart
	ChartDir = "jaegertracing/jaeger-operator"
	// ComponentServiceName is the name of the service.
	ComponentServiceName = "jaeger-operator-metrics"
	// ComponentWebhookServiceName is the name of the webhook service.
	ComponentWebhookServiceName = "jaeger-operator-webhook-service"
	// ComponentMutatingWebhookConfigName is the name of the mutating webhook config.
	ComponentMutatingWebhookConfigName = "jaeger-operator-mutating-webhook-configuration"
	// ComponentValidatingWebhookConfigName is the name of the mutating webhook config.
	ComponentValidatingWebhookConfigName = "jaeger-operator-validating-webhook-configuration"
	// ComponentCertificateName is the name of the Certificate.
	ComponentCertificateName = "jaeger-operator-serving-cert"
	// ComponentSecretName  is the name of the secret.
	ComponentSecretName = "jaeger-operator-service-cert"
	// JaegerCollectorDeploymentName is the name of the Jaeger instance collector deployment.
	JaegerCollectorDeploymentName = globalconst.JaegerInstanceName + "-" + globalconst.JaegerCollectorComponentName
	// JaegerQueryDeploymentName is the name of the Jaeger instance query deployment.
	JaegerQueryDeploymentName = globalconst.JaegerInstanceName + "-" + globalconst.JaegerQueryComponentName
)

type jaegerOperatorComponent struct {
	helm.HelmComponent
}

var (
	certificates = []types.NamespacedName{
		{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      jaegerCertificateName,
		},
	}
	jaegerIngressNames = []types.NamespacedName{
		{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      constants.JaegerIngress,
		},
	}
)

func NewComponent() spi.Component {
	return jaegerOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ChartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    "image.imagePullSecrets[0]",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "jaeger-operator-values.yaml"),
			Dependencies:              []string{certmanager.ComponentName, opensearch.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled returns true only if the Jaeger Operator is explicitly enabled
// in the Verrazzano CR.
func (c jaegerOperatorComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsJaegerOperatorEnabled(effectiveCR)
}

// IsReady checks if the Jaeger Operator deployment is ready
func (c jaegerOperatorComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		defaultJaegerEnabled, err := isJaegerCREnabled(ctx)
		if err == nil && defaultJaegerEnabled {
			return isJaegerOperatorReady(ctx) && isDefaultJaegerInstanceReady(ctx)
		}
		return isJaegerOperatorReady(ctx)
	}
	return false
}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
func (c jaegerOperatorComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.JaegerOperator == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.JaegerOperator.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.JaegerOperator.MonitorChanges
	}
	return true
}

// PreInstall updates resources necessary for the Jaeger Operator Component installation
func (c jaegerOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}

// PostInstall creates the ingress resource for exposing Jaeger UI service.
func (c jaegerOperatorComponent) PostInstall(ctx spi.ComponentContext) error {
	if err := c.createOrUpdateJaegerResources(ctx); err != nil {
		return err
	}
	// these need to be set for helm component post install processing
	c.IngressNames = c.GetIngressNames(ctx)
	c.Certificates = c.GetCertificateNames(ctx)
	return c.HelmComponent.PostInstall(ctx)
}

// PostUpgrade creates or updates the ingress of Jaeger UI service after a Verrazzano upgrade
func (c jaegerOperatorComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return c.createOrUpdateJaegerResources(ctx)
}

// ValidateInstall validates the installation of the Verrazzano CR
func (c jaegerOperatorComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	convertedVZ := installv1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(vz, &convertedVZ); err != nil {
		return err
	}
	if err := c.HelmComponent.ValidateInstallV1Beta1(&convertedVZ); err != nil {
		return err
	}
	return c.validateJaegerOperator(&convertedVZ)
}

// ValidateUpdate validates if the update operation of the Verrazzano CR is valid or not.
func (c jaegerOperatorComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	convertedVZNew := installv1beta1.Verrazzano{}
	convertedVZOld := installv1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(new, &convertedVZNew); err != nil {
		return err
	}
	if err := common.ConvertVerrazzanoCR(old, &convertedVZOld); err != nil {
		return err
	}
	if err := c.HelmComponent.ValidateUpdateV1Beta1(&convertedVZOld, &convertedVZNew); err != nil {
		return err
	}
	return c.validateJaegerOperator(&convertedVZNew)
}

// ValidateInstallV1Beta1 validates the installation of the Verrazzano CR
func (c jaegerOperatorComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	if err := c.HelmComponent.ValidateInstallV1Beta1(vz); err != nil {
		return err
	}
	return c.validateJaegerOperator(vz)
}

// ValidateUpdateV1Beta1 validates if the update operation of the Verrazzano CR is valid or not.
func (c jaegerOperatorComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	if err := c.HelmComponent.ValidateUpdateV1Beta1(old, new); err != nil {
		return err
	}
	return c.validateJaegerOperator(new)
}

// PreUpgrade Jaeger component pre-upgrade processing
func (c jaegerOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Jaeger pre-upgrade")
	// Create the verrazzano-monitoring namespace if not already created
	if err := common.EnsureVerrazzanoMonitoringNamespace(ctx); err != nil {
		return err
	}
	installed, err := helmcli.IsReleaseInstalled(ComponentName, ComponentNamespace)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed searching for Jaeger release: %v", err)
	}
	if !installed && doDefaultJaegerInstanceDeploymentsExists(ctx) {
		return ctx.Log().ErrorfNewErr("Conflicting Jaeger instance %s/%s exists! Either disable the Verrazzano's default Jaeger instance creation by overriding jaeger.create Helm value for Jaeger Operator component to false or delete and recreate the existing Jaeger deployment in a different namespace: %v", ComponentNamespace, globalconst.JaegerInstanceName, err)
	}
	err = removeOldJaegerResources(ctx)
	if err != nil {
		return err
	}
	createInstance, err := isCreateDefaultJaegerInstance(ctx)
	if err != nil {
		return err
	}
	if createInstance {
		// Create Jaeger secret with the OpenSearch credentials
		return createJaegerSecret(ctx)
	}
	return nil
}

// Upgrade jaegeroperator component for upgrade processing.
func (c jaegerOperatorComponent) Upgrade(ctx spi.ComponentContext) error {
	return c.HelmComponent.Install(ctx)
}

// IsInstalled checks if jaeger is installed
func (c jaegerOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", ComponentNamespace, ComponentName, err)
		return false, err
	}
	return true, nil
}

// GetIngressNames returns the Jaeger ingress name if Jaeger instance is enabled, otherwise returns
// an empty slice
func (c jaegerOperatorComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	if jaegerInstanceEnabled, _ := isJaegerCREnabled(ctx); jaegerInstanceEnabled {
		return jaegerIngressNames
	}
	return []types.NamespacedName{}
}

// GetCertificateNames returns the Jaeger certificate names if Jaeger instance is enabled, otherwise returns
// an empty slice
func (c jaegerOperatorComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	if jaegerInstanceEnabled, _ := isJaegerCREnabled(ctx); jaegerInstanceEnabled {
		return certificates
	}
	return []types.NamespacedName{}
}

// createOrUpdateJaegerResources create or update related Jaeger resources
func (c jaegerOperatorComponent) createOrUpdateJaegerResources(ctx spi.ComponentContext) error {
	jaegerCREnabled, err := isJaegerCREnabled(ctx)
	if err != nil {
		return err
	}
	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) && jaegerCREnabled {
		if err := createOrUpdateJaegerIngress(ctx, constants.VerrazzanoSystemNamespace); err != nil {
			return err
		}
	}
	return nil
}
