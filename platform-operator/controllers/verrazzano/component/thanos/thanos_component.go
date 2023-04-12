// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package thanos

import (
	"fmt"
	"path/filepath"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	promoperator "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus/operator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
			Dependencies:              []string{networkpolicies.ComponentName, nginx.ComponentName, promoperator.ComponentName},
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
func (t ThanosComponent) IsReady(ctx spi.ComponentContext) bool {
	return t.HelmComponent.IsReady(ctx) && t.isThanosReady(ctx)
}

// isThanosReady returns true if the availability objects that exist, have the minimum number of expected replicas
func (t ThanosComponent) isThanosReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	// If a Thanos subcomponent is enabled, the deployment or statefulset will definitely exist
	// once the Helm install completes. For the Thanos deployments and statefulsets that exist,
	// check if replicas are ready.
	deploymentsToCheck := []types.NamespacedName{}
	statefulsetsToCheck := []types.NamespacedName{}
	for _, deploymentName := range t.AvailabilityObjects.DeploymentNames {
		if exists, err := ready.DoesDeploymentExist(ctx.Client(), deploymentName); err == nil && exists {
			deploymentsToCheck = append(deploymentsToCheck, deploymentName)
		}
	}
	for _, stsName := range t.AvailabilityObjects.StatefulsetNames {
		if exists, err := ready.DoesStatefulsetExist(ctx.Client(), stsName); err == nil && exists {
			statefulsetsToCheck = append(statefulsetsToCheck, stsName)
		}
	}
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), deploymentsToCheck, 1, prefix) &&
		ready.StatefulSetsAreReady(ctx.Log(), ctx.Client(), statefulsetsToCheck, 1, prefix)
}

// IsEnabled Thanos enabled check for installation
func (t ThanosComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsThanosEnabled(effectiveCR)
}

// PreInstall handles the pre-install operations for the Thanos component
func (t ThanosComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return t.HelmComponent.PreInstall(ctx)
}

// PreUpgrade handles the pre-upgrade operations for the Thanos component
func (t ThanosComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}

	return t.HelmComponent.PreUpgrade(ctx)
}

// GetIngressNames returns the Thanos ingress names
func (t ThanosComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
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
	return append(ingressNames, types.NamespacedName{
		Namespace: ns,
		Name:      vzconst.ThanosQueryStoreIngress,
	})
}

// GetCertificateNames returns the TLS secret for the Thanos component
func (t ThanosComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
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
	return append(certificateNames, types.NamespacedName{
		Namespace: ns,
		Name:      queryStoreCertificateName,
	})
}
