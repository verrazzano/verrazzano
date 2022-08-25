// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"sigs.k8s.io/yaml"
)

// appendVerrazzanoOverrides appends the overrides for verrazzano component
func appendVerrazzanoOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	effectiveCR := ctx.EffectiveCR()
	// Find any storage overrides for the VMI, and
	resourceRequestOverrides, err := common.FindStorageOverride(effectiveCR)
	if err != nil {
		return kvs, err
	}

	// Overrides object to store any user overrides
	overrides := verrazzanoValues{}

	// Append the simple overrides
	if err := appendVerrazzanoValues(ctx, &overrides); err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed appending Verrazzano values: %v", err)
	}
	// Append any VMI overrides to the override values object, and any installArgs overrides to the kvs list
	vzkvs, err := appendVMIOverrides(effectiveCR, &overrides, resourceRequestOverrides, []bom.KeyValue{})
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed appending Verrazzano OpenSearch values: %v", err)
	}

	// append the security role overrides
	if err := appendSecurityOverrides(effectiveCR, &overrides); err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed appending Verrazzano security overrides: %v", err)
	}

	// Append any installArgs overrides to the kvs list
	vzkvs = appendVerrazzanoComponentOverrides(effectiveCR, vzkvs)

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed generating Verrazzano overrides file: %v", err)
	}

	// Append any installArgs overrides in vzkvs after the file overrides to ensure precedence of those
	kvs = append(kvs, bom.KeyValue{Value: overridesFileName, IsFile: true})
	kvs = append(kvs, vzkvs...)
	return kvs, nil
}

func generateOverridesFile(ctx spi.ComponentContext, overrides *verrazzanoValues) (string, error) {
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

func appendVerrazzanoValues(ctx spi.ComponentContext, overrides *verrazzanoValues) error {
	effectiveCR := ctx.EffectiveCR()

	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
	}

	if externalDNSEnabled := vzconfig.IsExternalDNSEnabled(effectiveCR); externalDNSEnabled {
		overrides.Externaldns = &externalDNSValues{
			Enabled: externalDNSEnabled,
		}
	}

	envName := vzconfig.GetEnvName(effectiveCR)
	overrides.Config = &configValues{
		EnvName:   envName,
		DNSSuffix: dnsSuffix,
	}

	overrides.Istio = &istioValues{Enabled: vzconfig.IsIstioEnabled(effectiveCR)}
	overrides.Keycloak = &keycloakValues{Enabled: vzconfig.IsKeycloakEnabled(effectiveCR)}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	overrides.PrometheusOperator = &prometheusOperatorValues{Enabled: vzconfig.IsPrometheusOperatorEnabled(effectiveCR)}
	overrides.PrometheusAdapter = &prometheusAdapterValues{Enabled: vzconfig.IsPrometheusAdapterEnabled(effectiveCR)}
	overrides.KubeStateMetrics = &kubeStateMetricsValues{Enabled: vzconfig.IsKubeStateMetricsEnabled(effectiveCR)}
	overrides.PrometheusPushgateway = &prometheusPushgatewayValues{Enabled: vzconfig.IsPrometheusPushgatewayEnabled(effectiveCR)}
	overrides.PrometheusNodeExporter = &prometheusNodeExporterValues{Enabled: vzconfig.IsPrometheusNodeExporterEnabled(effectiveCR)}
	overrides.JaegerOperator = &jaegerOperatorValues{Enabled: vzconfig.IsJaegerOperatorEnabled(effectiveCR)}
	return nil
}

func appendSecurityOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues) error {
	vzSpec := effectiveCR.Spec

	numAdminSubjects := len(vzSpec.Security.AdminSubjects)
	numMonSubjects := len(vzSpec.Security.MonitorSubjects)
	if numMonSubjects == 0 && numAdminSubjects == 0 {
		return nil
	}

	overrides.Security = &securityRoleBindingValues{}

	if numAdminSubjects > 0 {
		adminSubjectsMap := make(map[string]subject, numAdminSubjects)
		for i, adminSubj := range vzSpec.Security.AdminSubjects {
			subjectName := fmt.Sprintf("subject-%d", i)
			if err := vzconfig.ValidateRoleBindingSubject(adminSubj, subjectName); err != nil {
				return err
			}
			adminSubjectsMap[subjectName] = subject{
				Name:      adminSubj.Name,
				Kind:      adminSubj.Kind,
				Namespace: adminSubj.Namespace,
				APIGroup:  adminSubj.APIGroup,
			}
		}
		overrides.Security.AdminSubjects = adminSubjectsMap
	}
	if numMonSubjects > 0 {
		monSubjectMap := make(map[string]subject, numMonSubjects)
		for i, monSubj := range vzSpec.Security.MonitorSubjects {
			subjectName := fmt.Sprintf("subject-%d", i)
			if err := vzconfig.ValidateRoleBindingSubject(monSubj, fmt.Sprintf("monitorSubjects[%d]", i)); err != nil {
				return err
			}
			monSubjectMap[subjectName] = subject{
				Name:      monSubj.Name,
				Kind:      monSubj.Kind,
				Namespace: monSubj.Namespace,
				APIGroup:  monSubj.APIGroup,
			}
		}
		overrides.Security.MonitorSubjects = monSubjectMap
	}
	return nil
}

// appendVerrazzanoComponentOverrides - append overrides specified for the Verrazzano component
func appendVerrazzanoComponentOverrides(effectiveCR *vzapi.Verrazzano, kvs []bom.KeyValue) []bom.KeyValue {
	if effectiveCR.Spec.Components.Verrazzano != nil {
		for _, arg := range effectiveCR.Spec.Components.Verrazzano.InstallArgs {
			kvs = append(kvs, bom.KeyValue{
				Key:   arg.Name,
				Value: arg.Value,
			})
		}
	}
	return kvs
}

func appendVMIOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues, storageOverrides *common.ResourceRequestValues, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	overrides.Kibana = &kibanaValues{Enabled: vzconfig.IsOpenSearchDashboardsEnabled(effectiveCR)}

	overrides.ElasticSearch = &elasticsearchValues{
		Enabled: vzconfig.IsOpenSearchEnabled(effectiveCR),
	}
	multiNodeCluster, err := common.IsMultiNodeOpenSearch(effectiveCR)
	if err != nil {
		return kvs, err
	}
	overrides.ElasticSearch.MultiNodeCluster = multiNodeCluster

	overrides.Prometheus = &prometheusValues{
		Enabled:  vzconfig.IsPrometheusEnabled(effectiveCR),
		Requests: storageOverrides,
	}

	overrides.Grafana = &grafanaValues{
		Enabled:  vzconfig.IsGrafanaEnabled(effectiveCR),
		Requests: storageOverrides,
	}
	return kvs, nil
}
