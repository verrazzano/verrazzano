// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"context"
	"fmt"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"path/filepath"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ComponentName is the name of the component
const ComponentName = cmconstants.CertManagerComponentName

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.CertManagerNamespace

// ComponentJSONName is the JSON name of the verrazzano component in CRD
const ComponentJSONName = cmconstants.CertManagerComponentJSONName

// ExternalDNSComponentJSONName is the JSON name of the verrazzano component in CRD
//const ExternalDNSComponentJSONName = cmcommon.ExternalDNSComponentJSONName

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
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      certManagerDeploymentName,
						Namespace: ComponentNamespace,
					},
					{
						Name:      cainjectorDeploymentName,
						Namespace: ComponentNamespace,
					},
					{
						Name:      webhookDeploymentName,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsCertManagerEnabled(effectiveCR)
}

// IsReady component check
func (c certManagerComponent) IsReady(ctx spi.ComponentContext) bool {
	if ctx.IsDryRun() {
		ctx.Log().Debug("cert-manager PostUninstall dry run")
		return true
	}
	if c.HelmComponent.IsReady(ctx) {
		return c.isCertManagerReady(ctx)
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

// ValidateInstallV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be installed
func (c certManagerComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	if !c.IsEnabled(vz) {
		return nil
	}

	// Verify there isn't a CertManager installation that already exists in the cert-manager namespace
	if err := checkExistingCertManager(vz); err != nil {
		return err
	}

	return c.HelmComponent.ValidateInstallV1Beta1(vz)
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (c certManagerComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
}

// PreInstall runs before cert-manager components are installed
// The cert-manager namespace is created
// The cert-manager manifest is patched if needed and applied to create necessary CRDs
func (c certManagerComponent) PreInstall(compContext spi.ComponentContext) error {
	//vz := compContext.EffectiveCR()
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
	return c.HelmComponent.PreInstall(compContext)
}

// PostUninstall removes cert-manager objects that are created outside of Helm
func (c certManagerComponent) PostUninstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PostUninstall dry run")
		return nil
	}
	return uninstallCertManager(compContext)
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

func checkExistingCertManager(_ runtime.Object) error {
	// Check if the cert-manager namespace already exists and is not owned by Verrazzano
	client, err := k8sutil.GetCoreV1Func()
	if err != nil {
		return err
	}
	ns, err := client.Namespaces().Get(context.TODO(), ComponentNamespace, metav1.GetOptions{})
	if err != nil {
		if !kerrs.IsNotFound(err) {
			return err
		}
		return nil
	}
	if err = common.CheckExistingNamespace([]v1.Namespace{*ns}, func(namespace *v1.Namespace) bool {
		if namespace.Name == ComponentNamespace || namespace.Namespace == ComponentNamespace {
			return true
		}
		return false
	}); err != nil {
		return err
	}

	return nil
}
