// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerconfig

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	kerrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ComponentName is the name of the component
const ComponentName = "cert-manager-config"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.CertManagerNamespace

// ComponentJSONName - this is not a real component but declare it for compatiblity
const ComponentJSONName = "certManagerConfig"

// certManagerConfigComponent represents an CertManager component
type certManagerConfigComponent struct {
	helm.HelmComponent
}

// Verify that certManagerConfigComponent implements Component
var _ spi.Component = certManagerConfigComponent{}

// NewComponent returns a new CertManager component
func NewComponent() spi.Component {
	return certManagerConfigComponent{
		helm.HelmComponent{
			ReleaseName: ComponentName,
			JSONName:    ComponentJSONName,
			//ChartDir:                  filepath.Join(config.GetThirdPartyDir(), "cert-manager-config"),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			//ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			MinVerrazzanoVersion: constants.VerrazzanoVersion1_0_0,
			Dependencies:         []string{networkpolicies.ComponentName, certmanager.ComponentName},
		},
	}
}

// IsEnabled returns true if the cert-manager-config is enabled, which is the default
func (c certManagerConfigComponent) IsEnabled(_ runtime.Object) bool {
	return true
}

// IsReady component check
func (c certManagerConfigComponent) IsReady(ctx spi.ComponentContext) bool {
	return c.isCertManagerReady(ctx)
}

//// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
//func (c certManagerConfigComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
//	oldBeta := &v1beta1.Verrazzano{}
//	newBeta := &v1beta1.Verrazzano{}
//	if err := old.ConvertTo(oldBeta); err != nil {
//		return err
//	}
//	if err := new.ConvertTo(newBeta); err != nil {
//		return err
//	}
//
//	// Validate DNS updates only when there's a change in configuration
//	oldDNSName, _ := getDNSSuffix(oldBeta)
//	newDNSName, _ := getDNSSuffix(newBeta)
//	if oldDNSName != newDNSName || getEnvironmentName(oldBeta) != getEnvironmentName(newBeta) {
//		if err := validateLongestHostName(newBeta); err != nil {
//			return err
//		}
//	}
//	return c.ValidateUpdateV1Beta1(oldBeta, newBeta)
//}
//
//// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
//func (c certManagerConfigComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
//	vzV1Beta1 := &v1beta1.Verrazzano{}
//	if err := vz.ConvertTo(vzV1Beta1); err != nil {
//		return err
//	}
//	if err := validateLongestHostName(vz); err != nil {
//		return err
//	}
//	return c.ValidateInstallV1Beta1(vzV1Beta1)
//}
//
//// ValidateInstall checks if the specified new Verrazzano CR is valid for this component to be installed
//func (c certManagerConfigComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
//	if err := checkExistingCertManager(vz); err != nil {
//		return err
//	}
//	// Do not allow any changes except to enable the component post-install
//	if c.IsEnabled(vz) {
//		if _, err := validateConfiguration(vz.Spec.Components.CertManager); err != nil {
//			return err
//		}
//	}
//	if err := validateLongestHostName(vz); err != nil {
//		return err
//	}
//	return c.HelmComponent.ValidateInstallV1Beta1(vz)
//}
//
//// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
//func (c certManagerConfigComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
//	// Do not allow any changes except to enable the component post-install
//	if c.IsEnabled(old) && !c.IsEnabled(new) {
//		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
//	}
//	if _, err := validateConfiguration(new.Spec.Components.CertManager); err != nil {
//		return err
//	}
//
//	// Validate DNS updates only when there's a change in configuration
//	oldDNSName, _ := getDNSSuffix(old)
//	newDNSName, _ := getDNSSuffix(new)
//	if oldDNSName != newDNSName || getEnvironmentName(old) != getEnvironmentName(new) {
//		if err := validateLongestHostName(new); err != nil {
//			return err
//		}
//	}
//	return c.HelmComponent.ValidateUpdateV1Beta1(old, new)
//}

// PreInstall runs before cert-manager-config component is executed
func (c certManagerConfigComponent) PreInstall(compContext spi.ComponentContext) error {
	if err := c.checkExistingCertManager(compContext); err != nil {
		return err
	}
	if err := common.ProcessAdditionalCertificates(compContext.Log(), compContext.Client(), compContext.EffectiveCR()); err != nil {
		return err
	}
	return nil
}

func (c certManagerConfigComponent) Install(context spi.ComponentContext) error {
	// Nothing to install, eventually perhaps move to a chart or a different controller
	return nil
}

// PostInstall applies necessary cert-manager-config resources after the install has occurred
// In the case of an Acme cert, we install Acme resources
// In the case of a CA cert, we install CA resources
func (c certManagerConfigComponent) PostInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config PostInstall dry run")
		return nil
	}
	return c.createOrUpdateClusterIssuer(compContext)
}

func (c certManagerConfigComponent) Upgrade(context spi.ComponentContext) error {
	return nil
}

// PostUpgrade applies necessary cert-manager-config resources after upgrade has occurred
// In the case of an Acme cert, we install/update Acme resources
// In the case of a CA cert, we install/update CA resources
func (c certManagerConfigComponent) PostUpgrade(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-config PostInstall dry run")
		return nil
	}
	if err := c.checkExistingCertManager(compContext); err != nil {
		return err
	}
	return c.createOrUpdateClusterIssuer(compContext)
}

func (c certManagerConfigComponent) Uninstall(context spi.ComponentContext) error {
	return nil
}

// PostUninstall removes cert-manager-config objects that are created outside of Helm
func (c certManagerConfigComponent) PostUninstall(compContext spi.ComponentContext) error {
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-configPostUninstall dry run")
		return nil
	}
	if err := c.checkExistingCertManager(compContext); err != nil {
		return err
	}
	return uninstallCertManager(compContext)
}

func (c certManagerConfigComponent) createOrUpdateClusterIssuer(compContext spi.ComponentContext) error {
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
func (c certManagerConfigComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	return false
}

func (c certManagerConfigComponent) checkExistingCertManager(compCtx spi.ComponentContext) error {
	vz := compCtx.EffectiveCR()
	if vzcr.IsCertManagerEnabled(vz) {
		return nil
	}

	client, err := k8sutil.GetAPIExtV1Client(compCtx.Log())
	if err != nil {
		return err
	}

	crdNames := []string{
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"certificates.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"issuers.cert-manager.io",
	}

	for _, crdName := range crdNames {
		_, err = client.CustomResourceDefinitions().Get(context.TODO(), crdName, metav1.GetOptions{})
		if err != nil {
			if kerrs.IsNotFound(err) {
				return compCtx.Log().ErrorfThrottledNewErr("CertManager custom resource %s not found in cluster", crdName)
			}
			compCtx.Log().Errorf("Unxpected error looking up CertManager custom resource %s in cluster", crdName)
			return err
		}
	}
	return nil
}
