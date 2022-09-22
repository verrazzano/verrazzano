// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package networkpolicies

import (
	"context"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"io/fs"
	"io/ioutil"
	netv1 "k8s.io/api/networking/v1"
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
	overrides.Prometheus = &prometheusValues{Enabled: vzconfig.IsPrometheusEnabled(effectiveCR)}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	return nil
}

// removeNetPolsFromVerrazzanoHelmRelease disassociates the network policies from the Verrazzano release
func removeNetPolsFromVerrazzanoHelmRelease(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

	// Loop through the namespaces that have Verrazzano network policies
	for _, ns := range netpolNamespaces {
		log.Progressf("Looking for network policies in namespace %s", ns)

		// Get all the network policies in the current namespace
		netpolList := netv1.NetworkPolicyList{}
		cli.List(context.TODO(), &netpolList, &clipkg.ListOptions{Namespace: ns})

		// Remove the helm annotations for each policy IF the netpol is part of the Verrazzano release
		for i, netpol := range netpolList.Items {
			annotations := netpol.GetAnnotations()
			if annotations == nil {
				continue
			}
			if annotations["meta.helm.sh/release-name"] != constants.Verrazzano {
				continue
			}
			log.Progressf("Disassociating network policy %s:%s from Verrazzano Helm release", netpol.Namespace, netpol.Name)
			netpolNsn := types.NamespacedName{Name: netpol.Name, Namespace: netpol.Namespace}
			objs := []clipkg.Object{&netpolList.Items[i]}
			if _, err := common.RemoveAllHelmAnnotationsAndLabels(cli, objs[0], netpolNsn); err != nil {
				return log.ErrorfNewErr("Failed disassociating network policy %s:%s from Verrazzano Helm release: %v", netpol.Namespace, netpol.Name, err)
			}
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
