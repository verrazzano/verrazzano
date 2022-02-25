// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/verrazzano/verrazzano/pkg/bom"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"sigs.k8s.io/yaml"
)

// appendVerrazzanoOverrides appends the image overrides for the monitoring-init-images subcomponent
func appendVerrazzanoOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

	// Append some custom image overrides
	// - use local KeyValues array to ensure we append those after the file override; typically won't matter with the
	//   way we implement Helm calls, but don't depend on that
	vzkvs, err := appendCustomImageOverrides(kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to append custom image overrides: %v", err)
	}

	effectiveCR := ctx.EffectiveCR()
	// Find any storage overrides for the VMI, and
	resourceRequestOverrides, err := findStorageOverride(effectiveCR)
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
	vzkvs = appendVMIOverrides(effectiveCR, &overrides, resourceRequestOverrides, vzkvs)

	// append any fluentd overrides
	appendFluentdOverrides(effectiveCR, &overrides)
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

func appendCustomImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}

	imageOverrides, err := bomFile.BuildImageOverrides("monitoring-init-images")
	if err != nil {
		return kvs, err
	}

	kvs = append(kvs, imageOverrides...)
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

	overrides.Keycloak = &keycloakValues{Enabled: vzconfig.IsKeycloakEnabled(effectiveCR)}
	overrides.Rancher = &rancherValues{Enabled: vzconfig.IsRancherEnabled(effectiveCR)}
	overrides.Console = &consoleValues{Enabled: vzconfig.IsConsoleEnabled(effectiveCR)}
	overrides.VerrazzanoOperator = &voValues{Enabled: isVMOEnabled(effectiveCR)}
	overrides.MonitoringOperator = &vmoValues{Enabled: isVMOEnabled(effectiveCR)}
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

func appendVMIOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues, storageOverrides *resourceRequestValues, kvs []bom.KeyValue) []bom.KeyValue {
	overrides.Kibana = &kibanaValues{Enabled: vzconfig.IsKibanaEnabled(effectiveCR)}

	overrides.ElasticSearch = &elasticsearchValues{
		Enabled: vzconfig.IsElasticsearchEnabled(effectiveCR),
	}
	if storageOverrides != nil {
		overrides.ElasticSearch.Nodes = &esNodes{
			// Only have to override the data node storage
			Data: &esNodeValues{
				Requests: storageOverrides,
			},
		}
	}
	if effectiveCR.Spec.Components.Elasticsearch != nil {
		for _, arg := range effectiveCR.Spec.Components.Elasticsearch.ESInstallArgs {
			kvs = append(kvs, bom.KeyValue{
				Key:   fmt.Sprintf(esHelmValuePrefixFormat, arg.Name),
				Value: arg.Value,
			})
		}
	}

	overrides.Prometheus = &prometheusValues{
		Enabled:  vzconfig.IsPrometheusEnabled(effectiveCR),
		Requests: storageOverrides,
	}

	overrides.Grafana = &grafanaValues{
		Enabled:  vzconfig.IsGrafanaEnabled(effectiveCR),
		Requests: storageOverrides,
	}
	return kvs
}

func appendFluentdOverrides(effectiveCR *vzapi.Verrazzano, overrides *verrazzanoValues) {
	overrides.Fluentd = &fluentdValues{
		Enabled: vzconfig.IsFluentdEnabled(effectiveCR),
	}

	fluentd := effectiveCR.Spec.Components.Fluentd
	if fluentd != nil {
		overrides.Logging = &loggingValues{}
		if len(fluentd.ElasticsearchURL) > 0 {
			overrides.Logging.ElasticsearchURL = fluentd.ElasticsearchURL
		}
		if len(fluentd.ElasticsearchSecret) > 0 {
			overrides.Logging.ElasticsearchSecret = fluentd.ElasticsearchSecret
		}
		if len(fluentd.ExtraVolumeMounts) > 0 {
			for _, vm := range fluentd.ExtraVolumeMounts {
				dest := vm.Source
				if vm.Destination != "" {
					dest = vm.Destination
				}
				readOnly := true
				if vm.ReadOnly != nil {
					readOnly = *vm.ReadOnly
				}
				overrides.Fluentd.ExtraVolumeMounts = append(overrides.Fluentd.ExtraVolumeMounts,
					volumeMount{Source: vm.Source, Destination: dest, ReadOnly: readOnly})
			}
		}
		// Overrides for OCI Logging integration
		if fluentd.OCI != nil {
			overrides.Fluentd.OCI = &ociLoggingSettings{
				DefaultAppLogID: fluentd.OCI.DefaultAppLogID,
				SystemLogID:     fluentd.OCI.SystemLogID,
				APISecret:       fluentd.OCI.APISecret,
			}
		}
	}

	// Force the override to be the internal ES secret if the legacy ES secret is being used.
	// This may be the case during an upgrade from a version that was not using the ES internal password for Fluentd.
	if overrides.Logging.ElasticsearchSecret == globalconst.LegacyElasticsearchSecretName {
		overrides.Logging.ElasticsearchSecret = globalconst.VerrazzanoESInternal
	}
}
