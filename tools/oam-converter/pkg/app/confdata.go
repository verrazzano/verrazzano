// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app

import (
	"encoding/json"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	coherence "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/coherenceresources"
	helidon "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/helidonresources"
	weblogic "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/weblogicresources"
	extract "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	"io/ioutil"
	"log"
	"sigs.k8s.io/yaml"
	"strings"
)

// contains function checks if the specifies string is present in array of strings
func contains(arr []string, target string) bool {
	for _, element := range arr {
		if element == target {
			return true
		}
	}
	return false
}

func ConfData() error {

	helidonWorkloads := []*vzapi.VerrazzanoHelidonWorkload{}
	coherenceWorkloads := []*vzapi.VerrazzanoCoherenceWorkload{}
	//Read app File
	appData, err := ioutil.ReadFile("../../examples/bobs-books/bobs-books-app.yaml")
	if err != nil {
		fmt.Println("Failed to read YAML file:", err)
		return err
	}

	//Read Comp file
	compData, err := ioutil.ReadFile("../../examples/bobs-books/bobs-books-comp.yaml")
	if err != nil {
		fmt.Println("Failed to read YAML file:", err)
		return err
	}
	//A map for app file
	appMap := make(map[string]interface{})

	// Unmarshal the OAM YAML input data into the map
	err = yaml.Unmarshal(appData, &appMap)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	workloadTraitMap, componentNames, ingressTraits, metricsTraits := extract.HandleYAMLStructurePanic(appMap)
	fmt.Print(ingressTraits)

	//Splitting up the comp file with "---" delimiter into multiple objects
	compStr := string(compData)
	compObjects := strings.Split(compStr, "---")

	//Array of components in comp file
	var components []map[string]interface{}

	for _, obj := range compObjects {
		var component map[string]interface{}
		err := yaml.Unmarshal([]byte(obj), &component)
		if err != nil {
			log.Fatalf("Failed to unmarshal YAML: %v", err)
		}
		components = append(components, component)
	}
	weblogicMap := make(map[string]*vzapi.VerrazzanoWebLogicWorkload)
	coherenceWorkloads, helidonWorkloads, weblogicMap, err = segregateWorkloads(weblogicMap, components, componentNames, helidonWorkloads, coherenceWorkloads)
	if err != nil {
		fmt.Printf("Failed to segregate: %v\n", err)
		return err
	}

	for _, trait := range metricsTraits {

		fmt.Printf("Trait API Version: %s\n", trait.APIVersion)
		fmt.Printf("Trait name: %s\n", trait.Name)
		//Put metricsTrait method
	}

	err = createResources(workloadTraitMap, weblogicMap, coherenceWorkloads, helidonWorkloads)
	if err != nil {
		return err
	}
	return nil
}

func segregateWorkloads(weblogicMap map[string]*vzapi.VerrazzanoWebLogicWorkload, components []map[string]interface{}, componentNames []string, helidonWorkloads []*vzapi.VerrazzanoHelidonWorkload, coherenceWorkloads []*vzapi.VerrazzanoCoherenceWorkload) ([]*vzapi.VerrazzanoCoherenceWorkload, []*vzapi.VerrazzanoHelidonWorkload, map[string]*vzapi.VerrazzanoWebLogicWorkload, error) {
	//A weblogic map with the key as component name and value as a VerrazzanoWeblogicWorkload struct

	for _, comp := range components {

		var name string

		kind := comp["spec"].(map[string]interface{})["workload"].(map[string]interface{})["kind"].(string)
		name = comp["metadata"].(map[string]interface{})["name"].(string)

		//Checking if the specific component name is present in the component names array
		//where component names array is the array of component names
		//which has ingress traits applied on it
		if contains(componentNames, name) {
			if kind == "VerrazzanoWebLogicWorkload" {

				workload := comp["spec"].(map[string]interface{})["workload"].(map[string]interface{})
				weblogicWorkload := &vzapi.VerrazzanoWebLogicWorkload{}
				workloadJSON, err := json.Marshal(workload)

				if err != nil {
					log.Fatalf("Failed to marshal trait: %v", err)

				}

				err = json.Unmarshal(workloadJSON, &weblogicWorkload)
				if err != nil {
					fmt.Printf("Failed to unmarshal: %v\n", err)
					return nil, nil, nil, err
				}

				//putting into map of workloads whose key is the component name and
				//value is the weblogic workload
				weblogicMap[name] = weblogicWorkload
			}
			if kind == "VerrazzanoHelidonWorkload" {
				//Appending the helidon workloads in the helidon workload array
				workload := comp["spec"].(map[string]interface{})["workload"].(map[string]interface{})
				helidonWorkload := &vzapi.VerrazzanoHelidonWorkload{}
				workloadJSON, err := json.Marshal(workload)

				if err != nil {
					log.Fatalf("Failed to marshal trait: %v", err)
				}

				err = json.Unmarshal(workloadJSON, &helidonWorkload)
				if err != nil {
					fmt.Printf("Failed to unmarshal: %v\n", err)
					return nil, nil, nil, err
				}

				helidonWorkloads = append(helidonWorkloads, helidonWorkload)
			}
			if kind == "VerrazzanoCoherenceWorkload" {

				//Appending the coherence workloads in the coherence workload array
				name = comp["metadata"].(map[string]interface{})["name"].(string)
				workload := comp["spec"].(map[string]interface{})["workload"].(map[string]interface{})
				coherenceWorkload := &vzapi.VerrazzanoCoherenceWorkload{}
				workloadJSON, err := json.Marshal(workload)

				if err != nil {
					log.Fatalf("Failed to marshal trait: %v", err)
				}

				err = json.Unmarshal(workloadJSON, &coherenceWorkload)
				if err != nil {
					fmt.Printf("Failed to unmarshal: %v\n", err)
					return nil, nil, nil, err
				}

				coherenceWorkloads = append(coherenceWorkloads, coherenceWorkload)

			}
		}
	}
	return coherenceWorkloads, helidonWorkloads, weblogicMap, nil
}

func createResources(workloadTraitMap map[string]*vzapi.IngressTrait, weblogicMap map[string]*vzapi.VerrazzanoWebLogicWorkload, coherenceWorkloads []*vzapi.VerrazzanoCoherenceWorkload, helidonWorkloads []*vzapi.VerrazzanoHelidonWorkload) error {
	//Create child resources of each ingress trait
	for key, value := range workloadTraitMap {

		//fmt.Printf("Trait name: %s\n", trait.Name)
		for name := range weblogicMap {
			if name == key {
				err := weblogic.CreateIngressChildResourcesFromWeblogic(key, value, weblogicMap[name])
				if err != nil {
					return err
				}
			}
		}
		for _, coherenceWorkload := range coherenceWorkloads {
			if coherenceWorkload.Name == key {
				err := coherence.CreateIngressChildResourcesFromCoherence(key, value, coherenceWorkload)
				if err != nil {
					return err
				}
			}
		}
		for _, helidonWorkload := range helidonWorkloads {
			if helidonWorkload.Name == key {
				err := helidon.CreateIngressChildResourcesFromHelidon(key, value, helidonWorkload)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
