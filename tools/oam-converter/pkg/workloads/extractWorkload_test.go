// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloads

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	reader "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/testdata"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestExtractWorkload(t *testing.T) {

	compMaps := []map[string]interface{}{}

	compConf, err := reader.ReadFromYAMLTemplate("testdata/template/helidon_workload.yaml")
	if err != nil {
		return
	}
	compMaps = append(compMaps, compConf)
	appConf, err := reader.ReadFromYAMLTemplate("testdata/template/app_conf.yaml")
	if err != nil {
		return
	}

	spec, found, err := unstructured.NestedMap(appConf, "spec")
	if !found || err != nil {
		t.Fatalf("spec doesn't exist or not in the specified type")
	}

	appComponents, found, err := unstructured.NestedSlice(spec, "components")

	if !found || err != nil {
		t.Fatalf("app components doesn't exist or not in the specified type")
	}
	ingressData := make(map[string]interface{})
	for _, component := range appComponents {
		componentMap := component.(map[string]interface{})
		componentTraits, ok := componentMap[consts.YamlTraits].([]interface{})
		if ok && len(componentTraits) > 0 {
			for _, trait := range componentTraits {
				traitMap := trait.(map[string]interface{})
				ingressData, found, err = unstructured.NestedMap(traitMap, "trait")
				if !found || err != nil {
					t.Fatalf("ingress trait doesn't exist or not in the specified type")

				}

			}
		}
	}
	jsonData, err := json.Marshal(ingressData)
	if err != nil {
		t.Fatalf("error in marshaling data %v", err)
	}
	ingressTrait := &vzapi.IngressTrait{}
	err = json.Unmarshal(jsonData, &ingressTrait)
	if err != nil {
		t.Fatalf("error in unmarshalling data %v", err)
	}
	// Test data: conversionComponents array
	conversionComponents := []*types.ConversionComponents{
		{
			AppNamespace:  "hello-helidon",
			AppName:       "hello-helidon",
			ComponentName: "hello-helidon-component",
			IngressTrait:  ingressTrait,
		},
	}

	// Call the function to test
	result, err := ExtractWorkload(compMaps, conversionComponents)

	// Assertions
	assert.NoError(t, err)

	assert.NotNil(t, result[0].Helidonworkload, "Helidon Workload should not be nil")

}
