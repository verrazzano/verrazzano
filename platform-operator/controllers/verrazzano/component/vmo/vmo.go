// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

type vmiValues struct {
	ElasticSearch *elasticsearchValues `json:"elasticSearch,omitempty"`
	Prometheus    *prometheusValues    `json:"prometheus,omitempty"`
	Grafana       *grafanaValues       `json:"grafana,omitempty"`
	Kibana        *kibanaValues        `json:"kibana,omitempty"`
}

type resourceRequestValues struct {
	Memory  string `json:"memory,omitempty"`
	Storage string `json:"storage"` // Empty string allowed
}

type esNodeValues struct {
	Replicas int                    `json:"replicas,omitempty"`
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type esNodes struct {
	Master *esNodeValues `json:"master,omitempty"`
	Data   *esNodeValues `json:"data,omitempty"`
	Ingest *esNodeValues `json:"ingest,omitempty"`
}

type elasticsearchValues struct {
	Enabled bool     `json:"enabled"` // Always write
	Nodes   *esNodes `json:"nodes,omitempty"`
}

type kibanaValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type prometheusValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

type grafanaValues struct {
	Enabled  bool                   `json:"enabled"` // Always write
	Requests *resourceRequestValues `json:"requests,omitempty"`
}

// isVmoReady checks to see if the VMO component is in ready state
func isVmoReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// appendVmoOverrides appends overrides for the VMO component
func appendVmoOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	vzkvs, err := appendInitImageOverrides(kvs)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to append monitoring init image overrides: %v", err)
	}

	// Find any storage overrides for the VMI
	effectiveCR := ctx.EffectiveCR()
	resourceRequestOverrides, err := findStorageOverride(effectiveCR)
	if err != nil {
		return kvs, err
	}

	// Append any VMI overrides to the override values object
	overrides := vmiValues{}
	vzkvs = appendVMIOverrides(effectiveCR, &overrides, resourceRequestOverrides, vzkvs)

	// Get the dnsSuffix override
	dnsSuffix, err := vzconfig.GetDNSSuffix(ctx.Client(), effectiveCR)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed getting DNS suffix: %v", err)
	}
	kvs = append(kvs, bom.KeyValue{Key: "config.dnsSuffix", Value: dnsSuffix})

	// Get the env name
	envName := vzconfig.GetEnvName(effectiveCR)
	kvs = append(kvs, bom.KeyValue{Key: "config.envName", Value: envName})

	kvs = append(kvs, vzkvs...)

	return kvs, nil
}

// append the monitoring-init-images overrides
func appendInitImageOverrides(kvs []bom.KeyValue) ([]bom.KeyValue, error) {
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

func findStorageOverride(effectiveCR *vzapi.Verrazzano) (*resourceRequestValues, error) {
	if effectiveCR == nil || effectiveCR.Spec.DefaultVolumeSource == nil {
		return nil, nil
	}
	defaultVolumeSource := effectiveCR.Spec.DefaultVolumeSource
	if defaultVolumeSource.EmptyDir != nil {
		return &resourceRequestValues{
			Storage: "",
		}, nil
	}
	if defaultVolumeSource.PersistentVolumeClaim != nil {
		pvcClaim := defaultVolumeSource.PersistentVolumeClaim
		storageSpec, found := vzconfig.FindVolumeTemplate(pvcClaim.ClaimName, effectiveCR.Spec.VolumeClaimSpecTemplates)
		if !found {
			return nil, fmt.Errorf("Failed, did not find matching storage volume template for claim %s", pvcClaim.ClaimName)
		}
		storageString := storageSpec.Resources.Requests.Storage().String()
		return &resourceRequestValues{
			Storage: storageString,
		}, nil
	}
	return nil, fmt.Errorf("Failed, unsupported volume source: %v", defaultVolumeSource)
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
				Key:   fmt.Sprintf("elasticSearch.%s", arg.Name),
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
