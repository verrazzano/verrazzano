// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"reflect"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ComponentName is the name of the component
	ComponentName = "opensearch"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// vzImagePullSecretKeyName is the Helm key name for the VZ chart image pull secret
	vzImagePullSecretKeyName = "global.imagePullSecrets[0]"

	// verrazzanoSecretName is the name of the VMI secret
	verrazzanoSecretName       = "verrazzano"
	verrazzanoBackupScrtName   = "verrazzano-backup"
	objectstoreAccessKey       = "object_store_access_key"
	objectstoreAccessSecretKey = "object_store_secret_key"

	// Certificate names
	osCertificateName  = "system-tls-es-ingest"
	osdCertificateName = "system-tls-kibana"
)

// ComponentJSONName is the josn name of the opensearch component in CRD
const ComponentJSONName = "elasticsearch"

type opensearchComponent struct{}

// GetDependencies returns the dependencies of the Opensearch component
func (o opensearchComponent) GetDependencies() []string {
	return []string{istio.ComponentName, nginx.ComponentName}
}

// GetMinVerrazzanoVersion returns the minimum Verrazzano version required by the Opensearch component
func (o opensearchComponent) GetMinVerrazzanoVersion() string {
	return constants.VerrazzanoVersion1_3_0
}

// GetJSONName returns the josn name of the Opensearch component in CRD
func (o opensearchComponent) GetJSONName() string {
	return ComponentJSONName
}

func (o opensearchComponent) IsOperatorInstallSupported() bool {
	return true
}

func (o opensearchComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return isOpensearchInstalled(ctx)
}

func (o opensearchComponent) Reconcile(ctx spi.ComponentContext) error {
	return nil
}

func NewComponent() spi.Component {
	return opensearchComponent{}
}

// PreInstall Opensearch component pre-install processing; create and label required namespaces, copy any
// required secrets
func (o opensearchComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := setupSharedVMIResources(ctx); err != nil {
		return err
	}
	ctx.Log().Debug("OpenSearch pre-install")
	if err := createAndLabelNamespaces(ctx); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating/labeling namespace %s for OpenSearch : %v", ComponentNamespace, err)
	}
	return nil
}

// Install Opensearch component install processing
func (o opensearchComponent) Install(ctx spi.ComponentContext) error {
	return createVMI(ctx)
}

// PreUpgrade Opensearch component pre-upgrade processing
func (o opensearchComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return opensearchPreUpgrade(ctx)
}

// InstallUpgrade Opensearch component upgrade processing
func (o opensearchComponent) Upgrade(ctx spi.ComponentContext) error {
	return createVMI(ctx)
}

// IsReady component check
func (o opensearchComponent) IsReady(ctx spi.ComponentContext) bool {
	return isOpensearchReady(ctx)
}

// PostInstall - post-install, clean up temp files
func (o opensearchComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Opensearch component post-upgrade")

	cleanTempFiles(ctx)

	// Check if the ingresses and certs are present
	prefix := fmt.Sprintf("Component %s", ComponentName)
	if !status.IngressesPresent(ctx.Log(), ctx.Client(), o.GetIngressNames(ctx), prefix) {
		return ctrlerrors.RetryableError{
			Source:    ComponentName,
			Operation: "Check if Ingresses are present",
		}
	}

	if readyStatus, certsNotReady := status.CertificatesAreReady(ctx.Client(), ctx.Log(), ctx.EffectiveCR(), o.GetCertificateNames(ctx)); !readyStatus {
		ctx.Log().Progressf("Certificates not ready for component %s: %v", ComponentName, certsNotReady)
		return ctrlerrors.RetryableError{
			Source:    ComponentName,
			Operation: "Check if certificates are ready",
		}
	}
	return nil
}

// PostUpgrade Opensearch post-upgrade processing
func (o opensearchComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Opensearch component post-upgrade")

	cleanTempFiles(ctx)

	// Check if the ingresses and certs are present
	prefix := fmt.Sprintf("Component %s", ComponentName)
	if !status.IngressesPresent(ctx.Log(), ctx.Client(), o.GetIngressNames(ctx), prefix) {
		return ctrlerrors.RetryableError{
			Source:    ComponentName,
			Operation: "Check if Ingresses are present",
		}
	}

	if readyStatus, certsNotReady := status.CertificatesAreReady(ctx.Client(), ctx.Log(), ctx.EffectiveCR(), o.GetCertificateNames(ctx)); !readyStatus {
		ctx.Log().Progressf("Certificates not ready for component %s: %v", ComponentName, certsNotReady)
		return ctrlerrors.RetryableError{
			Source:    ComponentName,
			Operation: "Check if certificates are ready",
		}
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
	if err := o.checkEnabled(old, new); err != nil {
		return err
	}
	// Reject any other edits except InstallArgs
	// Do not allow any updates to storage settings via the volumeClaimSpecTemplates/defaultVolumeSource
	if err := compareStorageOverrides(old, new); err != nil {
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

func compareStorageOverrides(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// compare the storage overrides and reject if the type or size is different
	oldSetting, err := findStorageOverride(old)
	if err != nil {
		return err
	}
	newSetting, err := findStorageOverride(new)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(oldSetting, newSetting) {
		return fmt.Errorf("Can not change volume settings for %s", ComponentJSONName)
	}
	return nil
}

func (o opensearchComponent) checkEnabled(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow disabling of any component post-install for now
	if vzconfig.IsElasticsearchEnabled(old) && !vzconfig.IsElasticsearchEnabled(new) {
		return fmt.Errorf("Disabling component elasticsearch not allowed")
	}
	if vzconfig.IsGrafanaEnabled(old) && !vzconfig.IsGrafanaEnabled(new) {
		return fmt.Errorf("Disabling component grafana not allowed")
	}
	if vzconfig.IsPrometheusEnabled(old) && !vzconfig.IsPrometheusEnabled(new) {
		return fmt.Errorf("Disabling component prometheus not allowed")
	}
	if vzconfig.IsKibanaEnabled(old) && !vzconfig.IsKibanaEnabled(new) {
		return fmt.Errorf("Disabling component kibana not allowed")
	}
	return nil
}

// GetIngressNames - gets the names of the ingresses associated with this component
func (o opensearchComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName

	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      constants.ElasticsearchIngress,
		})
	}

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		ingressNames = append(ingressNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      constants.KibanaIngress,
		})
	}

	return ingressNames
}

// GetCertificateNames - gets the names of the ingresses associated with this component
func (o opensearchComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	if vzconfig.IsElasticsearchEnabled(ctx.EffectiveCR()) {
		certificateNames = append(certificateNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      osCertificateName,
		})
	}

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		certificateNames = append(certificateNames, types.NamespacedName{
			Namespace: ComponentNamespace,
			Name:      osdCertificateName,
		})
	}

	return certificateNames
}
