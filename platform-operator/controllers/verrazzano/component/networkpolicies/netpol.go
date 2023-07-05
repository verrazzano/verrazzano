// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

import (
	"context"
	"io/fs"
	"os"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/externaldns"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

const (
	tmpFilePrefix        = "verrazzano-netpol-overrides-"
	tmpSuffix            = "yaml"
	tmpFileCreatePattern = tmpFilePrefix + "*." + tmpSuffix
	tmpFileCleanPattern  = tmpFilePrefix + ".*\\." + tmpSuffix

	keycloakMySQLNetPolicyName = "keycloak-mysql"
	podSelectorAppLabelName    = "app"
)

// netpolNamespaceNames specifies the NSNs that have network policies
var netpolNamespaceNames = []types.NamespacedName{
	{Namespace: vzconst.CertManagerNamespace, Name: "cert-manager"},
	{Namespace: vzconst.CertManagerNamespace, Name: "external-dns"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istio-ingressgateway"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istio-egressgateway"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "allow-same-namespace"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istiocoredns"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istiod-access"},
	{Namespace: vzconst.KeycloakNamespace, Name: "keycloak"},
	{Namespace: vzconst.KeycloakNamespace, Name: keycloakMySQLNetPolicyName},
	{Namespace: vzconst.MySQLOperatorNamespace, Name: "mysql-operator"},
	{Namespace: vzconst.RancherSystemNamespace, Name: "cattle-cluster-agent"},
	{Namespace: vzconst.RancherSystemNamespace, Name: "rancher"},
	{Namespace: vzconst.RancherSystemNamespace, Name: "rancher-webhook"},
	{Namespace: vzconst.VerrazzanoMonitoringNamespace, Name: "jaeger-collector"},
	{Namespace: vzconst.VerrazzanoMonitoringNamespace, Name: "jaeger-query"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-authproxy"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-console"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-application-operator"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-application-operator-webhook"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "oam-kubernetes-runtime"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-es-master"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-es-data"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-os-ingest"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-osd"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-grafana"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "weblogic-operator"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "coherence-operator"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "kibana"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "kiali"},
	{Namespace: constants.VeleroNameSpace, Name: "allow-same-namespace"},
	{Namespace: constants.VeleroNameSpace, Name: "velero"},
	{Namespace: vzconst.ArgoCDNamespace, Name: "argocd"},
	{Namespace: vzconst.VerrazzanoCAPINamespace, Name: "clusterAPI"},
}

var (
	// For Unit test purposes
	writeFileFunc = os.WriteFile
)

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

	overrides.AuthProxy = &authproxyValues{Enabled: vzcr.IsAuthProxyEnabled(effectiveCR)}
	overrides.ConsoleValues = &consoleValues{Enabled: vzcr.IsConsoleEnabled(effectiveCR)}
	overrides.ApplicationOperator = &appOperatorValues{Enabled: vzcr.IsApplicationOperatorEnabled(effectiveCR)}
	overrides.OAM = &oamValues{Enabled: vzcr.IsOAMEnabled(effectiveCR)}
	overrides.WeblogicOperator = &weblogicOperatorValues{Enabled: vzcr.IsWebLogicOperatorEnabled(effectiveCR)}
	overrides.CoherenceOperator = &coherenceOperatorValues{Enabled: vzcr.IsCoherenceOperatorEnabled(effectiveCR)}
	overrides.ClusterOperator = &clusterOperatorValues{Enabled: vzcr.IsClusterOperatorEnabled(effectiveCR)}
	overrides.CertManager = &certManagerValues{Enabled: vzcr.IsCertManagerEnabled(effectiveCR)}
	overrides.NGINX = &nginxValues{Enabled: vzcr.IsNGINXEnabled(effectiveCR), Namespace: nginxutil.IngressNGINXNamespace()}
	overrides.ElasticSearch = &elasticsearchValues{Enabled: vzcr.IsOpenSearchEnabled(effectiveCR)}
	overrides.Externaldns = &externalDNSValues{Enabled: vzcr.IsExternalDNSEnabled(effectiveCR), Namespace: externaldns.ResolveExernalDNSNamespace()}
	overrides.Grafana = &grafanaValues{Enabled: vzcr.IsGrafanaEnabled(effectiveCR)}
	overrides.Istio = &istioValues{Enabled: vzcr.IsIstioEnabled(effectiveCR)}
	overrides.JaegerOperator = &jaegerOperatorValues{Enabled: vzcr.IsJaegerOperatorEnabled(effectiveCR)}
	overrides.Keycloak = &keycloakValues{Enabled: vzcr.IsKeycloakEnabled(effectiveCR)}

	promEnable := vzcr.IsPrometheusEnabled(effectiveCR) || vzcr.IsPrometheusOperatorEnabled(effectiveCR)
	overrides.Prometheus = &prometheusValues{Enabled: promEnable}
	overrides.Rancher = &rancherValues{Enabled: vzcr.IsRancherEnabled(effectiveCR)}
	overrides.Velero = &veleroValues{Enabled: vzcr.IsVeleroEnabled(effectiveCR)}
	overrides.ArgoCD = &argoCDValues{Enabled: vzcr.IsArgoCDEnabled(effectiveCR)}
	overrides.ClusterAPI = &clusterAPIValues{Enabled: vzcr.IsClusterAPIEnabled(effectiveCR)}
	overrides.FluentOperator = &fluentOperatorValues{Enabled: vzcr.IsFluentOperatorEnabled(effectiveCR)}
	return nil
}

// removeResourcePolicyFromHelm associates network policies, that used to be verrazzano helm resources,
// to the verrazzano-network-policies release
func associateNetworkPoliciesWithHelm(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

	// specify helm release nsn
	releaseNsn := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}

	ensureIngressNGINXNamespace(ctx.EffectiveCR().ObjectMeta)

	// Loop through the all the network policies
	for _, nsn := range netpolNamespaceNames {
		// Get the policy
		netpol := netv1.NetworkPolicy{}
		err := cli.Get(context.TODO(), nsn, &netpol)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return log.ErrorfNewErr("Error getting NetworkPolicy %v: %v", nsn, err)
		}
		// Associate the policy with the verrazzano-network-policies helm chart
		netpolNsn := types.NamespacedName{Name: netpol.Name, Namespace: netpol.Namespace}
		annotations := netpol.GetAnnotations()
		if annotations["meta.helm.sh/release-name"] == ComponentName {
			continue
		}

		log.Progressf("Associating NetworkPolicy %v with the verrazzano-network-policies Helm release", netpolNsn)

		objs := []clipkg.Object{&netpol}
		if _, err := common.AssociateHelmObject(cli, objs[0], releaseNsn, netpolNsn, true); err != nil {
			return log.ErrorfNewErr("Failed associating NetworkPolicy %s:%s from verrazzano Helm release: %v", netpol.Namespace, netpol.Name, err)
		}
	}
	return nil
}

// removeResourcePolicyFromHelm removes the Helm resource annotation to get rid of "keep" policy
func removeResourcePolicyFromHelm(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

	ensureIngressNGINXNamespace(ctx.EffectiveCR().ObjectMeta)

	// Loop through the all the network policies
	for _, nsn := range netpolNamespaceNames {
		// Get the policy
		netpol := netv1.NetworkPolicy{}
		err := cli.Get(context.TODO(), nsn, &netpol)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return log.ErrorfNewErr("Error getting NetworkPolicy %v: %v", nsn, err)
		}
		// Remove the app.kubernetes.io/managed-by helm annotations for each policy IF the netpol is part of the Verrazzano release
		annotations := netpol.GetAnnotations()
		if annotations == nil {
			continue
		}
		_, ok := annotations["helm.sh/resource-policy"]
		if !ok {
			continue
		}
		log.Progress("Removing helm.sh/resource-policy annotations from all network policies in the verrazzano-network-policies Helm release")
		netpolNsn := types.NamespacedName{Name: netpol.Name, Namespace: netpol.Namespace}
		objs := []clipkg.Object{&netpol}
		if _, err := common.RemoveResourcePolicyAnnotation(cli, objs[0], netpolNsn); err != nil {
			return log.ErrorfNewErr("Failed disassociating NetworkPolicy %s:%s from Verrazzano Helm release: %v", netpol.Namespace, netpol.Name, err)
		}
	}

	return nil
}

// cleanTempFiles - Clean up the override temp files in the temp dir
func cleanTempFiles(ctx spi.ComponentContext) {
	if err := vzos.RemoveTempFiles(ctx.Log().GetZapLogger(), tmpFileCleanPattern); err != nil {
		ctx.Log().Errorf("Failed deleting temp files: %v", err)
	}
}

// fixKeycloakMySQLNetPolicy fixes the keycloak-mysql network policy after an upgrade. When we apply the new version
// of the network policy, the podSelector match labels are merged with the existing policy match labels, but the
// old label no longer exists on the new MySQL pod. This function removes that label matcher if it exists.
func fixKeycloakMySQLNetPolicy(ctx spi.ComponentContext) error {
	netpol := &netv1.NetworkPolicy{}
	nsn := types.NamespacedName{Namespace: constants.KeycloakNamespace, Name: keycloakMySQLNetPolicyName}
	if err := ctx.Client().Get(context.TODO(), nsn, netpol); err != nil {
		// policy might not exist, e.g. MC managed cluster
		if errors.IsNotFound(err) {
			return nil
		}
		return ctx.Log().ErrorfThrottledNewErr("Error getting NetworkPolicy %v: %v", nsn, err)
	}

	// If there aren't any label matchers, we're done
	if netpol.Spec.PodSelector.MatchLabels == nil {
		return nil
	}

	// If the podSelector has an "app" label matcher, remove it
	if _, exists := netpol.Spec.PodSelector.MatchLabels[podSelectorAppLabelName]; exists {
		delete(netpol.Spec.PodSelector.MatchLabels, podSelectorAppLabelName)
		if err := ctx.Client().Update(context.TODO(), netpol, &clipkg.UpdateOptions{}); err != nil {
			ctx.Log().Errorf("Error updating network policy %s/%s: %v", constants.KeycloakNamespace, keycloakMySQLNetPolicyName, err)
			return err
		}
	}

	return nil
}

// ensureIngressNGINXNamespace ensures that Ingress NGINX NS is on the list
func ensureIngressNGINXNamespace(meta metav1.ObjectMeta) {
	namespace := nginxutil.IngressNGINXNamespace()
	for _, nsn := range netpolNamespaceNames {
		if nsn.Namespace == namespace {
			return
		}
	}
	netpolNamespaceNames = append(netpolNamespaceNames, types.NamespacedName{Namespace: namespace, Name: "ingress-nginx-controller"})
	netpolNamespaceNames = append(netpolNamespaceNames, types.NamespacedName{Namespace: namespace, Name: "ingress-nginx-default-backend"})
}
