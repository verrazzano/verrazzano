// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentbitocilaoutput

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common/override"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const (
	ComponentName      = "fluentbit-oci-logging-analytics-output"
	ComponentJSONName  = "fluentbitOCILoggingAnalyticsOutput"
	ComponentNamespace = constants.VerrazzanoSystemNamespace
)

type fluentbitOCILoggingAnalyticsOutput struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return fluentbitOCILoggingAnalyticsOutput{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			InstallBeforeUpgrade:      true,
			AppendOverridesFunc:       nil,
			GetInstallOverridesFunc:   getOverrides,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			Dependencies:              []string{fluentoperator.ComponentName},
			MinVerrazzanoVersion:      constants.VerrazzanoVersion2_0_0,
		},
	}
}

func (c fluentbitOCILoggingAnalyticsOutput) PreInstall(ctx spi.ComponentContext) error {
	if err := copyOCIApiSecret(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

func (c fluentbitOCILoggingAnalyticsOutput) PreUpgrade(ctx spi.ComponentContext) error {
	return c.HelmComponent.PreUpgrade(ctx)
}

func (c fluentbitOCILoggingAnalyticsOutput) Reconcile(ctx spi.ComponentContext) error {
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		err = c.Install(ctx)
	}
	return err
}

func (c fluentbitOCILoggingAnalyticsOutput) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsFluentbitOCILoggingAnalyticsOutputEnabled(effectiveCR)
}

// GetOverrides returns install overrides for a component
func getOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.FluentbitOCILoggingAnalyticsOutput != nil {
			return effectiveCR.Spec.Components.FluentbitOCILoggingAnalyticsOutput.ValueOverrides
		}
		return []v1alpha1.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	if effectiveCR.Spec.Components.FluentbitOCILoggingAnalyticsOutput != nil {
		return effectiveCR.Spec.Components.FluentbitOCILoggingAnalyticsOutput.ValueOverrides
	}
	return []v1beta1.Overrides{}
}

func (c fluentbitOCILoggingAnalyticsOutput) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.FluentbitOCILoggingAnalyticsOutput != nil {
		if ctx.EffectiveCR().Spec.Components.FluentbitOCILoggingAnalyticsOutput.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.FluentbitOCILoggingAnalyticsOutput.MonitorChanges
		}
		return true
	}
	return false
}
func copyOCIApiSecret(ctx spi.ComponentContext) error {
	apiSecretName, err := common.GetApiSecretName(ctx)
	if err != nil {
		return err
	}
	err = common.CopySecret(ctx, apiSecretName, ComponentNamespace, "OCI API")
	if err != nil {
		// Do not return an error in case of IsNotFound, we'll handle it in component validation
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return nil
}

/*
func (c fluentbitOCILoggingAnalyticsOutput) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	var vzv1beta1 *v1beta1.Verrazzano
	err := common.ConvertVerrazzanoCR(vz, vzv1beta1)
	if err != nil {
		return err
	}
	return c.validateOCILogAnalyticsOverrides(vzv1beta1)
}

func (c fluentbitOCILoggingAnalyticsOutput) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	var vzv1beta1 *v1beta1.Verrazzano
	err := common.ConvertVerrazzanoCR(new, vzv1beta1)
	if err != nil {
		return err
	}
	return c.validateOCILogAnalyticsOverrides(vzv1beta1)
}
*/

func (c fluentbitOCILoggingAnalyticsOutput) validateOCILogAnalyticsOverrides(vz *v1beta1.Verrazzano) error {
	if !c.IsEnabled(vz) {
		return nil
	}
	client, err := getClient()
	if err != nil {
		return err
	}
	var authType interface{}
	overrides, err := override.GetInstallOverridesYAMLUsingClient(client, vz.Spec.Components.FluentbitOCILoggingAnalyticsOutput.ValueOverrides, vz.Namespace)
	for _, overrideYAML := range overrides {
		if strings.Contains(overrideYAML, "auth:") {
			authType, err = override.ExtractValueFromOverrideString(overrideYAML, common.AuthTypeKey)
			if err != nil {
				return err
			}
			break
		}
	}
	switch authType.(string) {
	case common.UserPrincipal:
		if err := checkAPISecret(overrides); err != nil {
			return err
		}
	case common.OKEWorkloadIdentity:
		if err := checkRegion(overrides); err != nil {
			return err
		}
	}
	return checkObjectStoreNamespaceProvided(overrides)
}

func checkRegion(overrides []string) error {
	for _, overrideYAML := range overrides {
		if strings.Contains(overrideYAML, fmt.Sprintf("%s:", "auth")) {
			region, err := override.ExtractValueFromOverrideString(overrideYAML, common.AuthRegionKey)
			if err != nil {
				return err
			}
			if region == nil {
				continue
			}
			if len(region.(string)) == 0 {
				return fmt.Errorf("region cannot be empty")
			}
			return nil
		}
	}
	return fmt.Errorf("could not find region in overrides")
}

func checkAPISecret(overrides []string) error {
	var apiSecret interface{}
	var err error
	client, err := getClient()
	if err != nil {
		return err
	}
	for _, overrideYAML := range overrides {
		if strings.Contains(overrideYAML, fmt.Sprintf("%s:", "auth")) {
			apiSecret, err = override.ExtractValueFromOverrideString(overrideYAML, common.AuthApiSecretKey)
			if err != nil {
				return err
			}
			if apiSecret != nil {
				break
			}
		}
	}

	if apiSecret == nil {
		return fmt.Errorf("api Secret isn't provided")
	}
	secret := &corev1.Secret{}
	if err := validators.GetInstallSecret(client, apiSecret.(string), secret); err != nil {
		return err
	}
	// validate config secret
	if err := validators.ValidateFluentdConfigData(secret); err != nil {
		return err
	}
	// Validate key data exists and is a valid pem format
	pemData, err := validators.ValidateSecretKey(secret, validators.FluentdOCISecretPrivateKeyEntry, nil)
	if err != nil {
		return err
	}
	if err := validators.ValidatePrivateKey(secret.Name, pemData); err != nil {
		return err
	}
	return nil
}

func checkObjectStoreNamespaceProvided(overrides []string) error {
	for _, overrideYAML := range overrides {
		if strings.Contains(overrideYAML, fmt.Sprintf("%s:", common.OCIObjectStoreNamespaceKey)) {
			objStore, err := override.ExtractValueFromOverrideString(overrideYAML, common.OCIObjectStoreNamespaceKey)
			if err != nil {
				return err
			}
			if objStore == nil {
				continue
			}
			if len(objStore.(string)) == 0 {
				return fmt.Errorf("object Storage Namespace cannot be empty")
			}
			return nil
		}
	}
	return fmt.Errorf("could not find Object Storage Namespace in overrides")
}

func getClient() (clipkg.Client, error) {
	runtimeConfig, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	return clipkg.New(runtimeConfig, clipkg.Options{Scheme: newScheme()})
}

// newScheme creates a new scheme that includes this package's object for use by client
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)
	return scheme
}
