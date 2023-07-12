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
	"log"
)

// ExtractTrait - Extract traits from the app map
func ExtractTrait(appMap map[string]interface{}) ([]*types.ConversionComponents, error) {
	conversionComponents := []*types.ConversionComponents{}

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
		return nil, errors.New("namespace key doesn't exist")
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
					return nil, errors.New("trait spec doesn't exist")
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
					}

					err = json.Unmarshal(traitJSON, ingressTrait)

					if err != nil {
						fmt.Printf("Failed to unmarshal trait: %v", err)
					}

					conversionComponents = append(conversionComponents, &types.ConversionComponents{
						AppNamespace:  appNamespace,
						AppName:       appName,
						ComponentName: componentMap["componentName"].(string),
						IngressTrait:  ingressTrait,
					})
				}
			}
		}
	}

	return conversionComponents, nil
}

// ExtractWorkload - Extract workload from comp map
func ExtractWorkload(components []map[string]interface{}, conversionComponents []*types.ConversionComponents) ([]*types.ConversionComponents, error) {
	weblogicMap := make(map[string]*vzapi.VerrazzanoWebLogicWorkload)
	for _, comp := range components {

		spec, found, err := unstructured.NestedMap(comp, "spec")
		if !found || err != nil {
			return nil, errors.New("spec key in a component doesn't exist")
		}
		workload, found, err := unstructured.NestedMap(spec, "workload")
		if !found || err != nil {
			return nil, errors.New("workload in a component doesn't exist")
		}
		kind, found, err := unstructured.NestedString(workload, "kind")
		if !found || err != nil {
			return nil, errors.New("workload kind in a component doesn't exist")
		}

		compMetadata, found, err := unstructured.NestedMap(comp, "metadata")
		if !found || err != nil {
			return nil, errors.New("component metadata doesn't exist")
		}
		name, found, err := unstructured.NestedString(compMetadata, "name")
		if !found || err != nil {
			return nil, errors.New("component name doesn't exist")
		}

		//Checking if the specific component name is present in the component names array
		//where component names array is the array of component names
		//which has ingress traits applied on it

		for i := range conversionComponents {
			if conversionComponents[i].ComponentName == name {
				// Assign the desired value to the specific element
				switch kind {
				case "VerrazzanoWebLogicWorkload":

					weblogicWorkload := &vzapi.VerrazzanoWebLogicWorkload{}
					workloadJSON, err := json.Marshal(workload)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)

					}

					err = json.Unmarshal(workloadJSON, &weblogicWorkload)
					if err != nil {
						fmt.Printf("Failed to unmarshal: %v\n", err)

					}

					//putting into map of workloads whose key is the component name and
					//value is the weblogic workload
					weblogicMap[name] = weblogicWorkload
					conversionComponents[i].WeblogicworkloadMap = weblogicMap

				case "VerrazzanoHelidonWorkload":
					//Appending the helidon workloads in the helidon workload array
					helidonWorkload := &vzapi.VerrazzanoHelidonWorkload{}
					workloadJSON, err := json.Marshal(workload)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(workloadJSON, &helidonWorkload)
					if err != nil {
						fmt.Printf("Failed to unmarshal: %v\n", err)

					}
					conversionComponents[i].Helidonworkload = helidonWorkload
				case "VerrazzanoCoherenceWorkload":

					//Appending the coherence workloads in the coherence workload array
					coherenceWorkload := &vzapi.VerrazzanoCoherenceWorkload{}
					workloadJSON, err := json.Marshal(workload)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(workloadJSON, &coherenceWorkload)
					if err != nil {
						fmt.Printf("Failed to unmarshal: %v\n", err)

					}
					conversionComponents[i].Coherenceworkload = coherenceWorkload
				}
				break
			}
		}
	}
	return conversionComponents, nil
}
