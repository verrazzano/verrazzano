// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/vzcr"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"

	"k8s.io/apimachinery/pkg/types"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
)

const (
	// ComponentName is the name of the component
	ComponentName = "opensearch"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// Certificate names
	osCertificateName = "system-tls-os-ingest"

	// fluentbitFilterAndParserTemplate is the template name that consists Fluentbit Filter and Parser resource for Opensearch.
	fluentbitFilterAndParserTemplate = "opensearch-filter-parser.yaml"
)

// ComponentJSONName is the JSON name of the opensearch component in CRD
const ComponentJSONName = "opensearch"

type opensearchComponent struct{}

// Namespace returns the component namespace
func (o opensearchComponent) Namespace() string {
	return ComponentNamespace
}

// ShouldInstallBeforeUpgrade returns true if component can be installed before upgrade is done
func (o opensearchComponent) ShouldInstallBeforeUpgrade() bool {
	return false
}

// GetDependencies returns the dependencies of the OpenSearch component
func (o opensearchComponent) GetDependencies() []string {
	return []string{networkpolicies.ComponentName, vmo.ComponentName, fluentoperator.ComponentName}
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the OpenSearch component
func (o opensearchComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_0_0
}

// GetJSONName returns the josn name of the OpenSearch component in CRD
func (o opensearchComponent) GetJSONName() string {
	return ComponentJSONName
}

// GetOverrides returns the Helm override sources for a component
func (o opensearchComponent) GetOverrides(object runtime.Object) interface{} {
	if _, ok := object.(*vzapi.Verrazzano); ok {
		return []vzapi.Overrides{}
	} else if _, ok := object.(*installv1beta1.Verrazzano); ok {
		return []installv1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

// MonitorOverrides indicates whether monitoring of Helm override sources is enabled for a component
func (o opensearchComponent) MonitorOverrides(_ spi.ComponentContext) bool {
	return true
}

func (o opensearchComponent) IsOperatorInstallSupported() bool {
	return true
}

func (o opensearchComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return doesOSExist(ctx), nil
}

func (o opensearchComponent) Reconcile(_ spi.ComponentContext) error {
	return nil
}

func NewComponent() spi.Component {
	return opensearchComponent{}
}

// PreInstall OpenSearch component pre-install processing; create and label required namespaces, copy any
// required secrets
func (o opensearchComponent) PreInstall(ctx spi.ComponentContext) error {
	// create or update  VMI secret
	if err := common.EnsureVMISecret(ctx.Client()); err != nil {
		return err
	}
	// create or update backup VMI secret
	if err := common.EnsureBackupSecret(ctx.Client()); err != nil {
		return err
	}
	ctx.Log().Debug("OpenSearch pre-install")
	if err := common.CreateAndLabelVMINamespaces(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating/labeling namespace %s for OpenSearch : %v", ComponentNamespace, err)
	}
	return nil
}

// Install OpenSearch component install processing
func (o opensearchComponent) Install(ctx spi.ComponentContext) error {
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

func (o opensearchComponent) IsOperatorUninstallSupported() bool {
	return false
}

func (o opensearchComponent) PreUninstall(context spi.ComponentContext) error {
	return nil
}

func (o opensearchComponent) Uninstall(context spi.ComponentContext) error {
	return nil
}

func (o opensearchComponent) PostUninstall(context spi.ComponentContext) error {
	if err := common.CreateOrDeleteFluentbitFilterAndParser(context, fluentbitFilterAndParserTemplate, ComponentNamespace, true); err != nil {
		return err
	}
	return nil
}

// PreUpgrade OpenSearch component pre-upgrade processing
func (o opensearchComponent) PreUpgrade(ctx spi.ComponentContext) error {
	// create or update  VMI secret
	return common.EnsureVMISecret(ctx.Client())
}

// Upgrade OpenSearch component upgrade processing
func (o opensearchComponent) Upgrade(ctx spi.ComponentContext) error {
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

func (o opensearchComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available vzapi.ComponentAvailability) {
	return nodesToObjectKeys(ctx.EffectiveCR()).IsAvailable(ctx.Log(), ctx.Client())
}

// IsReady component check
func (o opensearchComponent) IsReady(ctx spi.ComponentContext) bool {
	return isOSReady(ctx)
}

// PostInstall OpenSearch post-install processing
func (o opensearchComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("OpenSearch component post-upgrade")
	if err := common.CreateOrDeleteFluentbitFilterAndParser(ctx, fluentbitFilterAndParserTemplate, ComponentNamespace, false); err != nil {
		return err
	}
	return common.CheckIngressesAndCerts(ctx, o)
}

// PostUpgrade OpenSearch post-upgrade processing
func (o opensearchComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("OpenSearch component post-upgrade")
	if err := common.CheckIngressesAndCerts(ctx, o); err != nil {
		return err
	}
	return nil
}

// IsEnabled opensearch-specific enabled check for installation
func (o opensearchComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsOpenSearchEnabled(effectiveCR)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (o opensearchComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	v1beta1Old := &installv1beta1.Verrazzano{}
	v1beta1New := &installv1beta1.Verrazzano{}
	if err := old.ConvertTo(v1beta1Old); err != nil {
		return err
	}
	if err := new.ConvertTo(v1beta1New); err != nil {
		return err
	}
	return o.ValidateUpdateV1Beta1(v1beta1Old, v1beta1New)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (o opensearchComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	vzv1beta1 := &installv1beta1.Verrazzano{}
	if err := vz.ConvertTo(vzv1beta1); err != nil {
		return err
	}
	return o.ValidateInstallV1Beta1(vzv1beta1)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (o opensearchComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow disabling active components
	if err := o.isOpenSearchEnabled(old, new); err != nil {
		return err
	}
	// Reject any other edits except InstallArgs
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := common.CompareStorageOverridesV1Beta1(old, new, ComponentJSONName); err != nil {
		return err
	}
	// Reject edits that duplicate names of install args or node groups
	return validateNoDuplicatedConfiguration(new)
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (o opensearchComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	return validateNoDuplicatedConfiguration(vz)
}

// Name returns the component name
func (o opensearchComponent) Name() string {
	return ComponentName
}

func (o opensearchComponent) isOpenSearchEnabled(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if vzcr.IsOpenSearchEnabled(old) && !vzcr.IsOpenSearchEnabled(new) {
		return fmt.Errorf("Disabling component %s not allowed", ComponentJSONName)
	}
	return nil
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (o opensearchComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName

	if vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      constants.OpensearchIngress,
		})
	}

	return ingressNames
}

// GetCertificateNames - gets the names of the certificates associated with this component
func (o opensearchComponent) GetCertificateNames(_ spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{
		{
			Namespace: ComponentNamespace,
			Name:      osCertificateName,
		},
	}
}
