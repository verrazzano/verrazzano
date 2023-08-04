// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package traits

import (
	"encoding/json"
	"errors"
	"fmt"

	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ExtractTrait - Extract traits from the app map
func ExtractTrait(appMaps []map[string]interface{}) ([]*types.ConversionComponents, error) {
	conversionComponents := []*types.ConversionComponents{}
	for _, appMap := range appMaps {
		appMetadata, found, err := unstructured.NestedMap(appMap, "metadata")
		if !found || err != nil {
			return nil, errors.New("app metadata doesn't exist")

		}
		appName, found, err := unstructured.NestedString(appMetadata, "name")
		if !found || err != nil {
			return nil, errors.New("app name key doesn't exist")
		}

		appNamespace, found, err := unstructured.NestedString(appMetadata, "namespace")
		if !found || err != nil {
			if types.InputArgs.Namespace == "" {
				return nil, errors.New("namespace key doesn't exist, please enter in the YAML or CLI args")
			}
			appNamespace = types.InputArgs.Namespace
		}

		appSpec, found, err := unstructured.NestedMap(appMap, "spec")
		if !found || err != nil {
			return nil, errors.New("app spec doesn't exist")
		}

		appComponents, found, err := unstructured.NestedSlice(appSpec, "components")
		if !found || err != nil {
			return nil, errors.New("app components doesn't exist")
		}

		for _, component := range appComponents {
			componentMap := component.(map[string]interface{})
			componentTraits, ok := componentMap[consts.YamlTraits].([]interface{})
			if ok && len(componentTraits) > 0 {
				for _, trait := range componentTraits {
					traitMap := trait.(map[string]interface{})
					//traitSpec := traitMap[consts.TraitComponent].(map[string]interface{})
					traitSpec, found, err := unstructured.NestedMap(traitMap, "trait")
					if !found || err != nil {
						return nil, fmt.Errorf("trait spec doesn't exist")

					}

					traitKind, found, err := unstructured.NestedString(traitSpec, "kind")
					if !found || err != nil {
						return nil, errors.New("trait kind doesn't exist")
					}
					if traitKind == consts.IngressTrait {
						ingressTrait := &vzapi.IngressTrait{}
						traitJSON, err := json.Marshal(traitSpec)

						if err != nil {
							fmt.Printf("Failed to marshal trait: %v", err)
							return nil, err

						}

						err = json.Unmarshal(traitJSON, ingressTrait)

						if err != nil {
							fmt.Printf("Failed to unmarshal trait: %v", err)
							return nil, err
						}

						conversionComponents = append(conversionComponents, &types.ConversionComponents{
							AppNamespace:  appNamespace,
							AppName:       appName,
							ComponentName: componentMap["componentName"].(string),
							IngressTrait:  ingressTrait,
						})
					}
					if traitKind == consts.MetricsTrait {
						metricsTrait := &vzapi.MetricsTrait{}
						traitJSON, err := json.Marshal(traitSpec)

						if err != nil {
							fmt.Printf("Failed to marshal trait: %v", err)
							return nil, err
						}

						err = json.Unmarshal(traitJSON, metricsTrait)

						if err != nil {
							fmt.Printf("Failed to unmarshal trait: %v", err)
							return nil, err
						}

						conversionComponents = append(conversionComponents, &types.ConversionComponents{
							AppNamespace:  appNamespace,
							AppName:       appName,
							ComponentName: componentMap["componentName"].(string),
							MetricsTrait:  metricsTrait,
						})
					}
				}
			}
		}
	}

	return conversionComponents, nil
}
