// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

import (
	"context"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"io/fs"
	"io/ioutil"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
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

// netpolNamespaceNames specifies the NSNs that have network policies
var netpolNamespaceNames []types.NamespacedName = []types.NamespacedName{
	{Namespace: vzconst.CertManagerNamespace, Name: "cert-manager"},
	{Namespace: vzconst.CertManagerNamespace, Name: "external-dns"},
	{Namespace: constants.IngressNginxNamespace, Name: "ingress-nginx-controller"},
	{Namespace: constants.IngressNginxNamespace, Name: "ingress-nginx-default-backend"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istio-ingressgateway"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istio-egressgateway"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "allow-same-namespace"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istiocoredns"},
	{Namespace: vzconst.IstioSystemNamespace, Name: "istiod-access"},
	{Namespace: vzconst.KeycloakNamespace, Name: "keycloak"},
	{Namespace: vzconst.KeycloakNamespace, Name: "keycloak-mysql"},
	{Namespace: vzconst.MySQLOperatorNamespace, Name: "mysql-operator"},
	{Namespace: vzconst.RancherSystemNamespace, Name: "cattle-cluster-agent"},
	{Namespace: vzconst.RancherSystemNamespace, Name: "rancher"},
	{Namespace: vzconst.RancherSystemNamespace, Name: "rancher-webhook"},
	{Namespace: constants.VerrazzanoMonitoringNamespace, Name: "jaeger-collector"},
	{Namespace: constants.VerrazzanoMonitoringNamespace, Name: "jaeger-query"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-authproxy"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-console"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "verrazzano-application-operator"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "oam-kubernetes-runtime"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-es-master"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-es-data"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-es-ingest"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-kibana"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "vmi-system-grafana"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "weblogic-operator"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "coherence-operator"},
	{Namespace: vzconst.VerrazzanoSystemNamespace, Name: "kibana"},
}

var (
	// For Unit test purposes
	writeFileFunc = ioutil.WriteFile
)

//func resetWriteFileFunc() {
//	writeFileFunc = ioutil.WriteFile
//}

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

	promEnable := vzconfig.IsPrometheusEnabled(effectiveCR) || vzconfig.IsPrometheusOperatorEnabled(effectiveCR)
	overrides.Prometheus = &prometheusValues{Enabled: promEnable}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	return nil
}

// removeResourcePolicyFromHelm associates network policies, that used to be verrazzano helm resources,
// to the verrazzano-network-policies release
func associateNetworkPoliciesWithHelm(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

	// specify helm release nsn
	releaseNsn := types.NamespacedName{Name: ComponentName, Namespace: ComponentNamespace}

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

		log.Progress("Associating network policies with verrazzano-network-policies Helm release")

		objs := []clipkg.Object{&netpol}
		if _, err := common.AssociateHelmObject(cli, objs[0], releaseNsn, netpolNsn, true); err != nil {
			return log.ErrorfNewErr("Failed associating NetworkPolicy %s:%s from Verrazzano Helm release: %v", netpol.Namespace, netpol.Name, err)
		}
	}
	return nil
}

// removeResourcePolicyFromHelm removes the Helm resource annotation to get rid of "keep" policy
func removeResourcePolicyFromHelm(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

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
		log.Progress("Removing resourcce-policy %s:%s from Verrazzano Helm release")
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
