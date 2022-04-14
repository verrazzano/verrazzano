// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"sigs.k8s.io/yaml"
)

// appendOpensearchOverrides appends the image overrides for the monitoring-init-images subcomponent
func appendOpensearchOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {

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
	overrides := vmiValues{}

	// Append any VMI overrides to the override values object, and any installArgs overrides to the kvs list
	vzkvs = appendVMIOverrides(effectiveCR, &overrides, resourceRequestOverrides, vzkvs)

	// Write the overrides file to a temp dir and add a helm file override argument
	overridesFileName, err := generateOverridesFile(ctx, &overrides)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed generating Opensearch overrides file: %v", err)
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

func generateOverridesFile(ctx spi.ComponentContext, overrides *vmiValues) (string, error) {
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
	ctx.Log().Debugf("Opensearch overrides file %s contents: %s", overridesFileName, string(bytes))
	return overridesFileName, nil
}

func appendVMIOverrides(effectiveCR *vzapi.Verrazzano, overrides *vmiValues, storageOverrides *resourceRequestValues, kvs []bom.KeyValue) []bom.KeyValue {
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
