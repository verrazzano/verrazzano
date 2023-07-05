// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/vmo"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	networkv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "prometheus-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoMonitoringNamespace

// ComponentJSONName is the JSON name of the component in the CRD
const ComponentJSONName = "prometheusOperator"

const chartDir = "prometheus-community/kube-prometheus-stack"

const (
	prometheusHostName        = "prometheus.vmi.system"
	prometheusCertificateName = "system-tls-prometheus"

	istioPrometheus = "prometheus-server"
)

type prometheusComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return prometheusComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), chartDir),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:    "global.imagePullSecrets[0].name",
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "prometheus-operator-values.yaml"),
			// the dependency on the VMO is to ensure that a persistent volume is retained and the claim is released
			// so that persistent storage can be migrated to the new Prometheus
			Dependencies:            []string{networkpolicies.ComponentName, nginx.ComponentName, cmconstants.CertManagerComponentName, vmo.ComponentName},
			AppendOverridesFunc:     AppendOverrides,
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// IsEnabled returns true if the Prometheus Operator is enabled or if the component is not specified
// in the Verrazzano CR.
func (c prometheusComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzcr.IsPrometheusOperatorEnabled(effectiveCR)
}

// IsReady checks if the Prometheus Operator deployment is ready
func (c prometheusComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isPrometheusOperatorReady(ctx)
	}
	return false
}

func (c prometheusComponent) IsAvailable(ctx spi.ComponentContext) (reason string, available vzapi.ComponentAvailability) {
	listOptions, err := prometheusOperatorListOptions()
	if err != nil {
		return err.Error(), vzapi.ComponentUnavailable
	}
	return (&ready.AvailabilityObjects{DeploymentSelectors: []clipkg.ListOption{listOptions}}).IsAvailable(ctx.Log(), ctx.Client())
}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
func (c prometheusComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.PrometheusOperator == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.PrometheusOperator.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.PrometheusOperator.MonitorChanges
	}
	return true
}

// PreInstall updates resources necessary for the Prometheus Operator Component installation
func (c prometheusComponent) PreInstall(ctx spi.ComponentContext) error {
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}
	if err := deleteNetworkPolicy(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreInstall(ctx)
}

// PreUpgrade updates resources necessary for the Prometheus Operator Component installation
func (c prometheusComponent) PreUpgrade(ctx spi.ComponentContext) error {
	if err := deleteNetworkPolicy(ctx); err != nil {
		return err
	}
	if err := preInstallUpgrade(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PreUpgrade(ctx)
}

// PostInstall creates/updates associated resources after this component is installed
func (c prometheusComponent) PostInstall(ctx spi.ComponentContext) error {
	if err := postInstallUpgrade(ctx); err != nil {
		return err
	}

	// these need to be set for helm component post install processing
	c.IngressNames = c.GetIngressNames(ctx)
	c.Certificates = c.GetCertificateNames(ctx)

	return c.HelmComponent.PostInstall(ctx)
}

// PostInstall creates/updates associated resources after this component is upgraded
func (c prometheusComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := postInstallUpgrade(ctx); err != nil {
		return err
	}

	return c.HelmComponent.PostUpgrade(ctx)
}

// ValidateInstall verifies the installation of the Verrazzano object
func (c prometheusComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	convertedVZ := installv1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(vz, &convertedVZ); err != nil {
		return err
	}
	if err := checkExistingCNEPrometheus(vz); err != nil {
		return err
	}
	return c.validatePrometheusOperator(&convertedVZ)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (c prometheusComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	convertedVZ := installv1beta1.Verrazzano{}
	if err := common.ConvertVerrazzanoCR(new, &convertedVZ); err != nil {
		return err
	}
	return c.validatePrometheusOperator(&convertedVZ)
}

// ValidateInstall verifies the installation of the Verrazzano object
func (c prometheusComponent) ValidateInstallV1Beta1(vz *installv1beta1.Verrazzano) error {
	if err := checkExistingCNEPrometheus(vz); err != nil {
		return err
	}
	return c.validatePrometheusOperator(vz)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (c prometheusComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return c.validatePrometheusOperator(new)
}

// getIngressNames - gets the names of the ingresses associated with this component
func (c prometheusComponent) GetIngressNames(ctx spi.ComponentContext) []types.NamespacedName {
	var ingressNames []types.NamespacedName
	if !vzcr.IsPrometheusEnabled(ctx.EffectiveCR()) || !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return ingressNames
	}

	ns := constants.VerrazzanoSystemNamespace
	if vzcr.IsAuthProxyEnabled(ctx.EffectiveCR()) {
		ns = authproxy.ComponentNamespace
	}
	return append(ingressNames, types.NamespacedName{
		Namespace: ns,
		Name:      constants.PrometheusIngress,
	})
}

// getCertificateNames - gets the names of the TLS ingress certificates associated with this component
func (c prometheusComponent) GetCertificateNames(ctx spi.ComponentContext) []types.NamespacedName {
	var certificateNames []types.NamespacedName

	if !vzcr.IsPrometheusEnabled(ctx.EffectiveCR()) || !vzcr.IsNGINXEnabled(ctx.EffectiveCR()) {
		return certificateNames
	}
	ns := constants.VerrazzanoSystemNamespace
	if vzcr.IsAuthProxyEnabled(ctx.EffectiveCR()) {
		ns = authproxy.ComponentNamespace
	}
	return append(certificateNames, types.NamespacedName{
		Namespace: ns,
		Name:      prometheusCertificateName,
	})
}

// checkExistingCNEPrometheus checks if Prometheus is already installed
// OLCNE Istio module may have Prometheus installed in istio-system namespace
func checkExistingCNEPrometheus(vz runtime.Object) error {
	if !vzcr.IsPrometheusEnabled(vz) {
		return nil
	}
	if err := k8sutil.ErrorIfDeploymentExists(constants.IstioSystemNamespace, istioPrometheus); err != nil {
		return err
	}
	if err := k8sutil.ErrorIfServiceExists(constants.IstioSystemNamespace, istioPrometheus); err != nil {
		return err
	}
	return nil
}

// PostUninstall is the Prometheus Operator PostInstall SPI function
func (c prometheusComponent) PostUninstall(ctx spi.ComponentContext) error {
	// delete the legacy prometheus ingress
	ingress := &networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.PrometheusIngress,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	err := ctx.Client().Delete(context.TODO(), ingress)
	if err != nil && !errors.IsNotFound(err) {
		ctx.Log().Errorf("Error deleting legacy Prometheus ingress %s, %v", constants.PrometheusIngress, err)
		return err
	}
	return nil
}
