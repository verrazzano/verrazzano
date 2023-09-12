// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"path"
	"path/filepath"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
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
			Dependencies:              []string{networkpolicies.ComponentName, certmanager.ComponentName, nginx.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
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
	if err := common.ApplyCRDYaml(ctx, config.GetHelmOpenSearchOpChartsDir()); err != nil {
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
	if err := o.HelmComponent.Install(ctx); err != nil {
		return err
	}

	ctx.Log().Infof("Applying OpensearchCluster CR")

	args, err := buildArgsForOpenSearchCR(ctx)
	if err != nil {
		return err
	}
	// substitute template values to all files in the directory and apply the resulting YAML
	filePath := path.Join(config.GetThirdPartyManifestsDir(), "opensearch-operator/opensearch_cluster_cr.yaml")
	yamlApplier := k8sutil.NewYAMLApplier(ctx.Client(), "")
	err = yamlApplier.ApplyFT(filePath, args)

	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed to substitute template values for System Monitors: %v", err)
	}

	return nil
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
