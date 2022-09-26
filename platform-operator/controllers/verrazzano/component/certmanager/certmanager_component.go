// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.CertManagerNamespace

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
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), "cert-manager"),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "cert-manager-values.yaml"),
			AppendOverridesFunc:       AppendOverrides,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_0_0,
			Dependencies:              []string{networkpolicies.ComponentName},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerComponent) IsEnabled(effectiveCR runtime.Object) bool {
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
func (c certManagerComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	oldBeta := &v1beta1.Verrazzano{}
	newBeta := &v1beta1.Verrazzano{}
	if err := old.ConvertTo(oldBeta); err != nil {
		return err
	}
	if err := new.ConvertTo(newBeta); err != nil {
		return err
	}
	return c.ValidateUpdateV1Beta1(oldBeta, newBeta)
}

// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
func (c certManagerComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	vzV1Beta1 := &v1beta1.Verrazzano{}
	if err := vz.ConvertTo(vzV1Beta1); err != nil {
		return err
	}
	return c.ValidateInstallV1Beta1(vzV1Beta1)
}

// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
func (c certManagerComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(vz) {
		if _, err := validateConfiguration(vz.Spec.Components.CertManager); err != nil {
			return err
		}
	}
	return c.HelmComponent.ValidateInstallV1Beta1(vz)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c certManagerComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	if _, err := validateConfiguration(new.Spec.Components.CertManager); err != nil {
		return err
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// PreInstall runs before cert-manager components are installed
// The cert-manager namespace is created
// The cert-manager manifest is patched if needed and applied to create necessary CRDs
func (c certManagerComponent) PreInstall(compContext spi.ComponentContext) error {
	vz := compContext.EffectiveCR()
	cli := compContext.Client()
	log := compContext.Log()
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PreInstall dry run")
		return nil
	}

	// create cert-manager namespace
	log.Debug("Adding label needed by network policies to cert-manager namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), cli, &ns, func() error {
		return nil
	}); err != nil {
		return log.ErrorfNewErr("Failed to create or update the cert-manager namespace: %v", err)
	}

	// Apply the cert-manager manifest, patching if needed
	log.Debug("Applying cert-manager crds")
	err := c.applyManifest(compContext)
	if err != nil {
		return log.ErrorfNewErr("Failed to apply the cert-manager manifest: %v", err)
	}
	if err := common.ProcessAdditionalCertificates(log, cli, vz); err != nil {
		return err
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

// PostUninstall removes cert-manager objects that are created outside of Helm
func (c certManagerComponent) PostUninstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PostUninstall dry run")
		return nil
	}
	return uninstallCertManager(compContext)
}

func (c certManagerComponent) createOrUpdateClusterIssuer(compContext spi.ComponentContext) error {
	isCAValue, err := isCA(compContext)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to verify the config type: %v", err)
	}
	var opResult controllerutil.OperationResult
	if !isCAValue {
		// Create resources needed for Acme certificates
		if opResult, err = createOrUpdateAcmeResources(compContext); err != nil {
			return compContext.Log().ErrorfNewErr("Failed creating Acme resources: %v", err)
		}
	} else {
		// Create resources needed for CA certificates
		if opResult, err = createOrUpdateCAResources(compContext); err != nil {
			msg := fmt.Sprintf("Failed creating CA resources: %v", err)
			compContext.Log().Once(msg)
			return fmt.Errorf(msg)
		}
	}
	if opResult == controllerutil.OperationResultCreated {
		// We're in the initial install phase, and created the ClusterIssuer for the first time,
		// so skip the renewal checks
		compContext.Log().Oncef("Initial install, skipping certificate renewal checks")
		return nil
	}
	// CertManager configuration was updated, cleanup any old resources from previous configuration
	// and renew certificates against the new ClusterIssuer
	if err := cleanupUnusedResources(compContext, isCAValue); err != nil {
		return err
	}
	if err := checkRenewAllCertificates(compContext, isCAValue); err != nil {
		compContext.Log().Errorf("Error requesting certificate renewal: %s", err.Error())
		return err
	}
	return nil
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c certManagerComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.CertManager != nil {
		if ctx.EffectiveCR().Spec.Components.CertManager.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.CertManager.MonitorChanges
		}
		return true
	}
	return false
}
