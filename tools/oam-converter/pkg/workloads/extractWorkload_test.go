// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloads

import (
	"encoding/json"
	"errors"
	"fmt"
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

	appMaps := []map[string]interface{}{}

	appConf, err := reader.ReadFromYAMLTemplate("testdata/template/app_conf.yaml")
	if err != nil {
		return
	}
	appMaps = append(appMaps, appConf)
	spec, found, err := unstructured.NestedMap(appConf, "spec")
	if !found || err != nil {
		errors.New("app components doesn't exist")
	}

	appComponents, found, err := unstructured.NestedSlice(spec, "components")

	if !found || err != nil {
		errors.New("app components doesn't exist")
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
					fmt.Errorf("trait spec doesn't exist")

				}

			}
		}
	}
	jsonData, err := json.Marshal(ingressData)
	ingressTrait := &vzapi.IngressTrait{}
	err = json.Unmarshal(jsonData, &ingressTrait)

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

	compSpec, found, err := unstructured.NestedMap(compConf, "spec")
	if !found || err != nil {
		errors.New("spec key in a component doesn't exist or not found in the specified type")
	}
	compWorkload, found, err := unstructured.NestedMap(compSpec, "workload")
	if !found || err != nil {
		errors.New("workload in a component doesn't exist or not found in the specified type")
	}
	expectedHelidonWorkload := &unstructured.Unstructured{
		Object: compWorkload,
	}

	assert.Equal(t, expectedHelidonWorkload, result[0].Helidonworkload)

}
