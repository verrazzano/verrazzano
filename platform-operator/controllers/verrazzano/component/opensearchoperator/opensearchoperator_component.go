// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	cmconst "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// ComponentName is the name of the component
	ComponentName = "opensearch-operator"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoLoggingNamespace

	// ComponentJSONName is the json name of the opensearch-operator component in CRD
	ComponentJSONName = "opensearchOperator"
)

type opensearchOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return opensearchOperatorComponent{
		HelmComponent: helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "opensearch-operator-values.yaml"),
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    "manager.imagePullSecrets[0]",
			Dependencies:              []string{networkpolicies.ComponentName, cmconst.ClusterIssuerComponentName, nginx.ComponentName, vmo.ComponentName},
			AppendOverridesFunc:       appendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled returns true if the component is enabled for install
func (o opensearchOperatorComponent) IsEnabled(effectiveCr runtime.Object) bool {
	return vzcr.IsOpenSearchOperatorEnabled(effectiveCr)
}

// IsReady - component specific ready-check
func (o opensearchOperatorComponent) IsReady(context spi.ComponentContext) bool {
	return o.isReady(context)
}

func (o opensearchOperatorComponent) IsAvailable(context spi.ComponentContext) (string, v1alpha1.ComponentAvailability) {
	deployments := getEnabledDeployments(context)
	actualAvailabilityObjects := ready.AvailabilityObjects{
		DeploymentNames: deployments,
	}
	return actualAvailabilityObjects.IsAvailable(context.Log(), context.Client())
}

// GetCertificateNames returns the list of certificates for this component
func (o opensearchOperatorComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	if !vzcr.IsOpenSearchOperatorEnabled(ctx.EffectiveCR()) {
		return nil
	}
	return clusterCertificates
}

// PreInstall runs before component is installed
func (o opensearchOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

	log.Debugf("Creating verrazzano-logging namespace")
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}

	ns.Labels["verrazzano.io/namespace"] = ComponentNamespace
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), cli, &ns, func() error {
		return nil
	}); err != nil {
		return log.ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	log.Debugf("Applying opensearch-oeprator crds")
	if err := common.ApplyCRDYamlWithDirectoryName(ctx, config.GetHelmOpenSearchOpChartsDir(), "files"); err != nil {
		return err
	}

	if err := handleLegacyOpenSearch(ctx); err != nil {
		return err
	}
	log.Debugf("Merging security configs")
	if err := common.MergeSecretData(ctx, config.GetThirdPartyManifestsDir()); err != nil {
		return err
	}
	return o.HelmComponent.PreInstall(ctx)
}

func (o opensearchOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	log := ctx.Log()
	log.Debugf("Merging security configs")
	if err := common.MergeSecretData(ctx, config.GetThirdPartyManifestsDir()); err != nil {
		return err
	}
	return o.HelmComponent.PreUpgrade(ctx)
}

func (o opensearchOperatorComponent) Install(ctx spi.ComponentContext) error {
	return o.HelmComponent.Install(ctx)
}

// PostInstall runs after component is installed
func (o opensearchOperatorComponent) PostInstall(ctx spi.ComponentContext) error {
	if err := resetReclaimPolicy(ctx); err != nil {
		return err
	}
	return nil
}

func (o opensearchOperatorComponent) PreUninstall(ctx spi.ComponentContext) error {
	return o.deleteRelatedResource()
}

func (o opensearchOperatorComponent) Uninstall(ctx spi.ComponentContext) error {
	if err := o.areRelatedResourcesDeleted(); err != nil {
		return err
	}
	return o.HelmComponent.Uninstall(ctx)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (o opensearchOperatorComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.OpenSearchOperator != nil {
		if ctx.EffectiveCR().Spec.Components.OpenSearchOperator.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.OpenSearchOperator.MonitorChanges
		}
		return true
	}
	return false
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (o opensearchOperatorComponent) ValidateInstall(vz *v1alpha1.Verrazzano) error {
	vzv1beta1 := &v1beta1.Verrazzano{}
	if err := vz.ConvertTo(vzv1beta1); err != nil {
		return err
	}
	return o.ValidateInstallV1Beta1(vzv1beta1)
}

// ValidateInstallV1Beta1 checks if the specified Verrazzano CR is valid for this component to be installed
func (o opensearchOperatorComponent) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) error {
	if !o.IsEnabled(vz) {
		if vzcr.IsOpenSearchEnabled(vz) {
			return fmt.Errorf("Opensearch Operator must be enabled if Opensearch is enabled")
		}
		if vzcr.IsOpenSearchDashboardsEnabled(vz) {
			return fmt.Errorf("Opensearch Operator must be enabled if Opensearch Dashboards is enabled")
		}
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (o opensearchOperatorComponent) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) error {
	// do not allow disabling active components
	if vzcr.IsOpenSearchOperatorEnabled(old) && !vzcr.IsOpenSearchOperatorEnabled(new) {
		return fmt.Errorf("Disabling component %s not allowed", ComponentJSONName)
	}
	return nil
}

// ValidateUpdateV1Beta1 checks if the specified new Verrazzano CR is valid for this component to be updated
func (o opensearchOperatorComponent) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) error {
	// do not allow disabling active components
	if vzcr.IsOpenSearchOperatorEnabled(old) && !vzcr.IsOpenSearchOperatorEnabled(new) {
		return fmt.Errorf("Disabling component %s not allowed", ComponentJSONName)
	}
	return nil
}
