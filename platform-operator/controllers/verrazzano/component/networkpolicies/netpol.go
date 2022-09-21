// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/namespace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"io/fs"
	"io/ioutil"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
)

const (
	tmpFilePrefix        = "verrazzano-netpol-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix
)

// netpolNamespaces specifies the namespaces that have network policies
var netpolNamespaces []string = []string{
	vzconst.CertManagerNamespace,
	constants.IngressNginxNamespace,
	constants.IstioSystemNamespace,
	constants.KeycloakNamespace,
	vzconst.MySQLOperatorNamespace,
	vzconst.RancherSystemNamespace,
	constants.VerrazzanoMonitoringNamespace,
	constants.VerrazzanoSystemNamespace,
}

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

func resetWriteFileFunc() {
	writeFileFunc = ioutil.WriteFile
}

// getOverrides returns install overrides for a component
func getOverrides(object runtime.Object) interface{} {
	return []vzapi.Overrides{}
}

// appendOverrides appends the overrides for this component
func appendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Overrides object to store any user overrides
	overrides := chartValues{}

	// Append the simple overrides
	if err := appendVerrazzanoValues(ctx, &overrides); err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed appending Verrazzano values: %v", err)
	}

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed generating Verrazzano overrides file: %v", err)
	}

	// Append any installArgs overrides in vzkvs after the file overrides to ensure precedence of those
	vzkvs := append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	kvs = append(kvs, vzkvs...)
	return kvs, nil
}

func generateOverridesFile(ctx spi.ComponentContext, overrides *chartValues) (string, error) {
	bytes, err := yaml.Marshal(overrides)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp(os.TempDir(), tmpFileCreatePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, bytes, fs.ModeAppend); err != nil {
		return "", err
	}
	ctx.Log().Debugf("Verrazzano install overrides file %s contents: %s", overridesFileName, string(bytes))
	return overridesFileName, nil
}

func appendVerrazzanoValues(ctx spi.ComponentContext, overrides *chartValues) error {
	effectiveCR := ctx.EffectiveCR()

	overrides.ElasticSearch = &elasticsearchValues{Enabled: vzconfig.IsOpenSearchEnabled(effectiveCR)}
	overrides.Externaldns = &externalDNSValues{Enabled: vzconfig.IsExternalDNSEnabled(effectiveCR)}
	overrides.Grafana = &grafanaValues{Enabled: vzconfig.IsGrafanaEnabled(effectiveCR)}
	overrides.Istio = &istioValues{Enabled: vzconfig.IsIstioEnabled(effectiveCR)}
	overrides.JaegerOperator = &jaegerOperatorValues{Enabled: vzconfig.IsJaegerOperatorEnabled(effectiveCR)}
	overrides.Keycloak = &keycloakValues{Enabled: vzconfig.IsKeycloakEnabled(effectiveCR)}
	overrides.Prometheus = &prometheusValues{Enabled: vzconfig.IsPrometheusEnabled(effectiveCR)}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	return nil
}

// AssociateNetworkPoliciesWithHelmRelease associates the network policies with the verrazzano-network-policies helm chart
func AssociateNetworkPoliciesWithHelmRelease(cli clipkg.Client) error {
	return associateNetworkPolicies(cli, true)
}

func associateNetworkPolicies(cli clipkg.Client, keep bool) error {
	// specify helm release nsn
	releaseNsn := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}

	// Loop through the namespaces that have Verrazzano network policies
	for _, ns := range netpolNamespaces {
		// Get all the network policies in the current namespace
		netpolList := netv1.NetworkPolicyList{}
		cli.List(context.TODO(), &netpolList, &clipkg.ListOptions{Namespace: ns})

		// Associate each policy with the verrazzano-network-policies helm chart
		for _, netpol := range netpolList.Items {
			netpolNsn := types.NamespacedName{Name: netpol.Name, Namespace: netpol.Namespace}
			objs := []clipkg.Object{&netpol}
			if _, err := common.AssociateHelmObject(cli, objs[0], releaseNsn, netpolNsn, keep); err != nil {
				return err
			}
		}
	}
	return nil
}

// ensureNamespaces creates the netwprk policy namespaces if they don't exist
func ensureNamespaces(cli clipkg.Client, cr *v1beta1.Verrazzano) error {

	return nil
}

func CreateAndLabelNamespaces(ctx spi.ComponentContext) error {
	if err := LabelKubeSystemNamespace(ctx.Client()); err != nil {
		return err
	}
	if err := common.CreateAndLabelVMINamespaces(ctx); err != nil {
		return err
	}
	if err := namespace.CreateVerrazzanoMultiClusterNamespace(ctx.Client()); err != nil {
		return err
	}
	if vzconfig.IsKeycloakEnabled(ctx.EffectiveCR()) {
		istio := ctx.EffectiveCR().Spec.Components.Istio
		if err := namespace.CreateKeycloakNamespace(ctx.Client(), istio != nil && istio.IsInjectionEnabled()); err != nil {
			return ctx.Log().ErrorfNewErr("Failed creating Keycloak namespace: %v", err)
		}
	}
	// cattle-system NS must be created since the rancher NetworkPolicy, which is always installed, requires it
	if err := namespace.CreateRancherNamespace(ctx.Client()); err != nil {
		return ctx.Log().ErrorfNewErr("Failed creating Rancher namespace: %v", err)
	}
	return nil
}

// LabelKubeSystemNamespace adds the label needed by network polices to kube-system
func LabelKubeSystemNamespace(client clipkg.Client) error {
	const KubeSystemNamespace = "kube-system"
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: KubeSystemNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), client, &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = KubeSystemNamespace
		return nil
	}); err != nil {
		return err
	}
	return nil
}
