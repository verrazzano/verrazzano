// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package traits

import (
	"encoding/json"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"log"
)

// handling YAML structure panic
func HandleYAMLStructurePanic(yamlMap map[string]interface{}) (map[string]*vzapi.IngressTrait, []string, []*vzapi.IngressTrait, []*vzapi.MetricsTrait) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("panic occurred in YAML structure:", err)
		}
	}()
	return extractTraitFromMap(yamlMap)
}

// Extract traits from the component map
func extractTraitFromMap(yamlMap map[string]interface{}) (map[string]*vzapi.IngressTrait, []string, []*vzapi.IngressTrait, []*vzapi.MetricsTrait) {

	ingressTraits := []*vzapi.IngressTrait{} //Array of ingresstraits
	metricsTraits := []*vzapi.MetricsTrait{}

	// Create an ingress map
	ingressMap := make(map[string]*vzapi.IngressTrait)

	var componentNames []string

	// Access nested objects within the YAML data and extract traits by checking the kind of the object
	components := yamlMap[consts.YamlSpec].(map[string]interface{})[consts.YamlComponents].([]interface{})
	for _, component := range components {
		componentMap := component.(map[string]interface{})
		componentTraits, ok := componentMap[consts.YamlTraits].([]interface{})
		if ok && len(componentTraits) > 0 {
			for _, trait := range componentTraits {
				traitMap := trait.(map[string]interface{})
				traitSpec := traitMap[consts.TraitComponent].(map[string]interface{})
				traitKind := traitSpec[consts.TraitKind].(string)
				if traitKind == consts.IngressTrait {
					ingressTrait := &vzapi.IngressTrait{}
					traitJSON, err := json.Marshal(traitSpec)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(traitJSON, ingressTrait)
					ingressTrait.Name = yamlMap[consts.YamlMetadata].(map[string]interface{})[consts.YamlName].(string)
					ingressTrait.Namespace = yamlMap[consts.YamlMetadata].(map[string]interface{})[consts.YamlNamespace].(string)
					if err != nil {
						log.Fatalf("Failed to unmarshal trait: %v", err)
					}
					fmt.Println(componentMap["componentName"].(string))

					//Assigning ingresstrait to a map with the key as component name
					ingressMap[componentMap["componentName"].(string)] = ingressTrait

					componentNames = append(componentNames, componentMap["componentName"].(string))
					ingressTraits = append(ingressTraits, ingressTrait)
				}

				fmt.Println("check map", ingressMap[" robert-helidon"])
				if traitKind == consts.MetricsTrait {
					//	Add metricstrait code

				}
			}
		}
	}
	for _, name := range componentNames {
		fmt.Println("name of the component who has ingress trait", name)
	}
	return ingressMap, componentNames, ingressTraits, metricsTraits
}
