// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ComponentName is the name of the component
	ComponentName = "opensearch"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// Certificate names
	osCertificateName = "system-tls-es-ingest"
)

// ComponentJSONName is the josn name of the opensearch component in CRD
const ComponentJSONName = "opensearch"

type opensearchComponent struct{}

// GetDependencies returns the dependencies of the OpenSearch component
func (o opensearchComponent) GetDependencies() []string {
	return []string{istio.ComponentName, nginx.ComponentName, vmo.ComponentName}
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the OpenSearch component
func (o opensearchComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_3_0
}

// GetJSONName returns the josn name of the OpenSearch component in CRD
func (o opensearchComponent) GetJSONName() string {
	return ComponentJSONName
}

func (o opensearchComponent) IsOperatorInstallSupported() bool {
	return true
}

func (o opensearchComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return doesOSExist(ctx), nil
}

func (o opensearchComponent) Reconcile(ctx spi.ComponentContext) error {
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

// PreUpgrade OpenSearch component pre-upgrade processing
func (o opensearchComponent) PreUpgrade(ctx spi.ComponentContext) error {
	// create or update  VMI secret
	return common.EnsureVMISecret(ctx.Client())
}

// Upgrade OpenSearch component upgrade processing
func (o opensearchComponent) Upgrade(ctx spi.ComponentContext) error {
	return common.CreateOrUpdateVMI(ctx, updateFunc)
}

// IsReady component check
func (o opensearchComponent) IsReady(ctx spi.ComponentContext) bool {
	return checkOpenSearchStatus(ctx, status.DeploymentsAreReady, status.StatefulSetsAreReady)
}

// PostInstall OpenSearch post-install processing
func (o opensearchComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("OpenSearch component post-upgrade")
	return common.CheckIngressesAndCerts(ctx, o)
}

// PostUpgrade OpenSearch post-upgrade processing
func (o opensearchComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("OpenSearch component post-upgrade")
	if err := common.CheckIngressesAndCerts(ctx, o); err != nil {
		return err
	}
	return o.updateElasticsearchResources(ctx)
}

// updateElasticsearchResources updates elasticsearch resources
func (o opensearchComponent) updateElasticsearchResources(ctx spi.ComponentContext) error {
	if err := fixupElasticSearchReplicaCount(ctx, ComponentNamespace); err != nil {
		return err
	}
	return nil
}

// IsEnabled opensearch-specific enabled check for installation
func (o opensearchComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Elasticsearch
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (o opensearchComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling active components
	if err := o.isOpenSearchEnabled(old, new); err != nil {
		return err
	}
	// Reject any other edits except InstallArgs
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := common.CompareStorageOverrides(old, new, ComponentJSONName); err != nil {
		return err
	}
	return nil
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (o opensearchComponent) ValidateInstall(_ *vzapi.Verrazzano) error {
	return nil
}

// Name returns the component name
func (o opensearchComponent) Name() string {
	return ComponentName
}

func (o opensearchComponent) isOpenSearchEnabled(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if vzconfig.IsElasticsearchEnabled(old) && !vzconfig.IsElasticsearchEnabled(new) {
		return fmt.Errorf("Disabling component OpenSearch not allowed")
	}
	return nil
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (o opensearchComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{{
		Namespace: ComponentNamespace,
		Name:      constants.ElasticsearchIngress,
	}}
}

// GetCertificateNames - gets the names of the ingresses associated with this component
func (o opensearchComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	return []types.NamespacedName{{
		Namespace: ComponentNamespace,
		Name:      osCertificateName,
	}}
}
