// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentbitosoutput

import (
	"context"
	"net/url"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	ComponentName                  = "fluentbit-opensearch-output"
	ComponentJSONName              = "fluentbitOpensearchOutput"
	ComponentNamespace             = constants.VerrazzanoSystemNamespace
	OverrideApplicationHostKey     = "application.host"
	OverrideApplicationPasswordKey = "application.httpPassword.valueFrom.secretKeyRef.name"
	OverrideApplicationUserKey     = "application.httpUser.valueFrom.secretKeyRef.name"
	OverrideSystemHostKey          = "system.host"
	OverrideSystemPasswordKey      = "system.httpPassword.valueFrom.secretKeyRef.name"
	OverrideSystemUserKey          = "system.httpUser.valueFrom.secretKeyRef.name"
	OverrideApplicationPortKey     = "application.port"
	OverrideSystemPortKey          = "system.port"
	OverrideSystemTLSKey           = "system.tls.enabled"
	OverrideSystemCAFileKey        = "system.tls.caFile"
	OverrideSystemCertKey          = "system.tls.crtFile"
	OverrideApplicationTLSKey      = "application.tls.enabled"
	OverrideApplicationCAFileKey   = "application.tls.caFile"
	OverrideApplicationCertKey     = "application.tls.crtFile"
	FluentBitCertPath              = "/etc/ssl/certs/ca-bundle.crt"
	CACertPath                     = "/fluent-bit/etc/secret"
	CACertName                     = "ca-cert.crt"
)

type fluentbitOpensearchOutput struct {
	helm.HelmComponent
}

var _ spi.Component = fluentbitOpensearchOutput{}

func NewComponent() spi.Component {
	return fluentbitOpensearchOutput{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_6_0,
			GetInstallOverridesFunc:   getOverrides,
			Dependencies:              []string{fluentoperator.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			InstallBeforeUpgrade:      true,
		},
	}
}

func (c fluentbitOpensearchOutput) PreInstall(ctx spi.ComponentContext) error {
	if err := checkOpensearchSecretExists(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

func (c fluentbitOpensearchOutput) PreUpgrade(ctx spi.ComponentContext) error {
	if err := checkOpensearchSecretExists(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreUpgrade(ctx)
}

func (c fluentbitOpensearchOutput) Reconcile(ctx spi.ComponentContext) error {
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		err = c.Install(ctx)
	}
	return err
}

// GetOverrides returns install overrides for a component
func getOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.FluentbitOpensearchOutput != nil {
			return effectiveCR.Spec.Components.FluentbitOpensearchOutput.ValueOverrides
		}
		return []v1alpha1.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	if effectiveCR.Spec.Components.FluentbitOpensearchOutput != nil {
		return effectiveCR.Spec.Components.FluentbitOpensearchOutput.ValueOverrides
	}
	return []v1beta1.Overrides{}
}

func (c fluentbitOpensearchOutput) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.FluentbitOpensearchOutput != nil {
		if ctx.EffectiveCR().Spec.Components.FluentbitOpensearchOutput.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.FluentbitOpensearchOutput.MonitorChanges
		}
		return true
	}
	return false
}

func (c fluentbitOpensearchOutput) IsEnabled(cr runtime.Object) bool {
	return vzcr.IsFluentbitOpensearchOutputEnabled(cr)
}

// checkOpensearchSecretExists checks if secret with Opensearch Credential exists or not.
func checkOpensearchSecretExists(ctx spi.ComponentContext) error {
	if vzcr.IsKeycloakEnabled(ctx.EffectiveCR()) {
		secretName := globalconst.VerrazzanoESInternal
		secret := &corev1.Secret{}
		err := ctx.Client().Get(context.TODO(), clipkg.ObjectKey{
			Namespace: constants.VerrazzanoSystemNamespace,
			Name:      secretName,
		}, secret)
		if err != nil {
			if errors.IsNotFound(err) {
				ctx.Log().Progressf("Component FluentbitOpensearchOutput waiting for the secret %s/%s to exist",
					constants.VerrazzanoSystemNamespace, secretName)
				return ctrlerrors.RetryableError{Source: ComponentName}
			}
			ctx.Log().Errorf("Component FluentbitOpensearchOutput failed to get the secret %s/%s: %v",
				constants.VerrazzanoSystemNamespace, secretName, err)
			return err
		}
	}
	return nil
}

// AppendOverrides appends the Overrides for fluentbitOpensearchOutput.
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	registrationSecret, err := common.GetManagedClusterRegistrationSecret(ctx.Client())
	if err != nil {
		return kvs, err
	}
	if registrationSecret != nil {
		opensearchURL := string(registrationSecret.Data[constants.OpensearchURLData])
		kvs, err = getFluentOSOutputOverrides(opensearchURL, kvs)
		if err != nil {
			return kvs, err
		}
	}

	return kvs, nil
}

// getFluentOSOutputOverrides gets the Overrides for fluentbitOpensearchOutput in managed cluster.
func getFluentOSOutputOverrides(OpensearchURL string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// extracting the Hostname from the URL
	urlObject, err := url.Parse(OpensearchURL)
	if err != nil {
		return kvs, err
	}
	host := urlObject.Hostname()
	port := urlObject.Port()
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationHostKey, Value: host})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemHostKey, Value: host})
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationPortKey, Value: port})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemPortKey, Value: port})
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationPasswordKey, Value: constants.MCRegistrationSecret})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemPasswordKey, Value: constants.MCRegistrationSecret})
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationUserKey, Value: constants.MCRegistrationSecret})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemUserKey, Value: constants.MCRegistrationSecret})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemTLSKey, Value: "true"})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemCAFileKey, Value: CACertPath + "/" + CACertName})
	kvs = append(kvs, bom.KeyValue{Key: OverrideSystemCertKey, Value: FluentBitCertPath})
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationTLSKey, Value: "true"})
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationCAFileKey, Value: CACertPath + "/" + CACertName})
	kvs = append(kvs, bom.KeyValue{Key: OverrideApplicationCertKey, Value: FluentBitCertPath})
	return kvs, nil
}
