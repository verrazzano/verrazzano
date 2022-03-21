// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = "cert-manager"

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "certManager"

// certManagerComponent represents an CertManager component
type certManagerComponent struct {
	helm.HelmComponent
}

// Verify that certManagerComponent implements Component
var _ spi.Component = certManagerComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), "cert-manager"),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "cert-manager-values.yaml"),
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_0_0,
			Dependencies:            []string{},
		},
	}
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	return vzconfig.IsCertManagerEnabled(effectiveCR)
}

// IsReady component check
func (c certManagerComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isCertManagerReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c certManagerComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if !reflect.DeepEqual(c.getCertificateSettings(old), c.getCertificateSettings(new)) {
		return fmt.Errorf("Updates to certificate settings not allowed for %s", c.GetJSONName())
	}
	return nil
}

// PreInstall runs before cert-manager components are installed
// The cert-manager namespace is created
// The cert-manager manifest is patched if needed and applied to create necessary CRDs
func (c certManagerComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PreInstall dry run")
		return nil
	}

	// create cert-manager namespace
	compContext.Log().Debug("Adding label needed by network policies to cert-manager namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the cert-manager namespace: %v", err)
	}

	// Apply the cert-manager manifest, patching if needed
	compContext.Log().Debug("Applying cert-manager crds")
	err := c.applyManifest(compContext)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to apply the cert-manager manifest: %v", err)
	}
	return nil
}

// PostInstall applies necessary cert-manager resources after the install has occurred
// In the case of an Acme cert, we install Acme resources
// In the case of a CA cert, we install CA resources
func (c certManagerComponent) PostInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PostInstall dry run")
		return nil
	}
	return c.createOrUpdateClusterIssuer(compContext)
}

// PostUpgrade applies necessary cert-manager resources after upgrade has occurred
// In the case of an Acme cert, we install/update Acme resources
// In the case of a CA cert, we install/update CA resources
func (c certManagerComponent) PostUpgrade(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PostInstall dry run")
		return nil
	}
	return c.createOrUpdateClusterIssuer(compContext)
}

func (c certManagerComponent) getCertificateSettings(vz *vzapi.Verrazzano) vzapi.Certificate {
	var certSettings vzapi.Certificate
	if vz.Spec.Components.CertManager != nil {
		certSettings = vz.Spec.Components.CertManager.Certificate
	}
	return certSettings
}
