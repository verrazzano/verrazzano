// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "thanos"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the JSON name of the Thanos component in CRD
const ComponentJSONName = "thanos"

// Availability Object Names
const (
	queryDeployment         = "thanos-query"
	frontendDeployment      = "thanos-query-frontend"
	storeGatewayStatefulset = "thanos-storegateway"
)

type ThanosComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return ThanosComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ImagePullSecretKeyname:    "image.pullSecrets[0]",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "thanos-values.yaml"),
			Dependencies:              []string{common.IstioComponentName, networkpolicies.ComponentName, nginx.ComponentName, common.PrometheusOperatorComponentName, fluentoperator.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			GetInstallOverridesFunc:   GetOverrides,
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: []types.NamespacedName{
					{
						Name:      frontendDeployment,
						Namespace: ComponentNamespace,
					},
					{
						Name:      queryDeployment,
						Namespace: ComponentNamespace,
					},
				},
				StatefulsetNames: []types.NamespacedName{
					{
						Name:      storeGatewayStatefulset,
						Namespace: ComponentNamespace,
					},
				},
			},
		},
	}
}

// IsReady component check for Thanos
func (c ThanosComponent) IsReady(ctx spi.ComponentContext) bool {
	return c.HelmComponent.IsReady(ctx) && c.isThanosReady(ctx)
}

// IsAvailable returns the component availability for ThanosComponent, also accounting for optional
// subcomponents like store gateway
func (c ThanosComponent) IsAvailable(ctx spi.ComponentContext) (string, v1alpha1.ComponentAvailability) {
	deployments := c.getEnabledDeployments(ctx)
	statefulsets := c.getEnabledStatefulsets(ctx)
	actualAvailabilityObjects := ready.AvailabilityObjects{
		DeploymentNames:  deployments,
		StatefulsetNames: statefulsets,
	}
	return actualAvailabilityObjects.IsAvailable(ctx.Log(), ctx.Client())
}

// isThanosReady returns true if the availability objects that exist, have the minimum number of expected replicas
func (c ThanosComponent) isThanosReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	// If a Thanos subcomponent is enabled, the deployment or statefulset will definitely exist
	// once the Helm install completes. For the Thanos deployments and statefulsets that exist,
	// check if replicas are ready.
	deploymentsToCheck := c.getEnabledDeployments(ctx)
	statefulsetsToCheck := c.getEnabledStatefulsets(ctx)

	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deploymentsToCheck, 1, prefix) &&
		ready.StatefulSetsAreReady(ctx.Log(), ctx.Client(), statefulsetsToCheck, 1, prefix)
}

func (c ThanosComponent) getEnabledDeployments(ctx spi.ComponentContext) []types.NamespacedName {
	enabledDeployments := []types.NamespacedName{}
	for _, deploymentName := range c.AvailabilityObjects.DeploymentNames {
		if exists, err := ready.DoesDeploymentExist(ctx.Client(), deploymentName); err == nil && exists {
			enabledDeployments = append(enabledDeployments, deploymentName)
		}
	}
	return enabledDeployments
}

func (c ThanosComponent) getEnabledStatefulsets(ctx spi.ComponentContext) []types.NamespacedName {
	enabledStatefulsets := []types.NamespacedName{}
	for _, stsName := range c.AvailabilityObjects.StatefulsetNames {
		if exists, err := ready.DoesStatefulsetExist(ctx.Client(), stsName); err == nil && exists {
			enabledStatefulsets = append(enabledStatefulsets, stsName)
		}
	}
	return enabledStatefulsets
}

// IsEnabled Thanos enabled check for installation
func (c ThanosComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsThanosEnabled(effectiveCR)
}

// PreInstall handles the pre-install operations for the Thanos component
func (c ThanosComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return c.HelmComponent.PreInstall(ctx)
}

// PreUpgrade handles the pre-upgrade operations for the Thanos component
func (c ThanosComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return c.HelmComponent.PreUpgrade(ctx)
}

// GetIngressNames returns the Thanos ingress names
func (c ThanosComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName
	if !vzcr.IsThanosEnabled(ctx.EffectiveCR()) || !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return ingressNames
	}
	ns := constants.VerrazzanoSystemNamespace
	if vzcr.IsAuthProxyEnabled(ctx.EffectiveCR()) {
		ns = authproxy.ComponentNamespace
	}
	ingressNames = append(ingressNames, types.NamespacedName{
		Namespace: ns,
		Name:      vzconst.ThanosQueryIngress,
	})
	ingressNames = append(ingressNames, types.NamespacedName{
		Namespace: ns,
		Name:      vzconst.ThanosQueryStoreIngress,
	})
	return append(ingressNames, types.NamespacedName{
		Namespace: ns,
		Name:      vzconst.ThanosRulerIngress,
	})
}

// GetCertificateNames returns the TLS secret for the Thanos component
func (c ThanosComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	if !vzcr.IsThanosEnabled(ctx.EffectiveCR()) || !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return certificateNames
	}
	ns := constants.VerrazzanoSystemNamespace
	if vzcr.IsAuthProxyEnabled(ctx.EffectiveCR()) {
		ns = authproxy.ComponentNamespace
	}
	certificateNames = append(certificateNames, types.NamespacedName{
		Namespace: ns,
		Name:      queryCertificateName,
	})
	certificateNames = append(certificateNames, types.NamespacedName{
		Namespace: ns,
		Name:      queryStoreCertificateName,
	})
	return append(certificateNames, types.NamespacedName{
		Namespace: ns,
		Name:      rulerCertificateName,
	})
}
