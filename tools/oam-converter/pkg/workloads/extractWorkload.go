// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloads

import (
	"encoding/json"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
)

// ExtractWorkload - Extract workload from comp map
func ExtractWorkload(components []map[string]interface{}, conversionComponents []*types.ConversionComponents) ([]*types.ConversionComponents, error) {
	weblogicMap := &unstructured.Unstructured{}
	for _, comp := range components {

		spec, found, err := unstructured.NestedMap(comp, "spec")
		if !found || err != nil {
			return nil, errors.New("spec key in a component doesn't exist or not found in the specified type")
		}
		workload, found, err := unstructured.NestedMap(spec, "workload")
		if !found || err != nil {
			return nil, errors.New("workload in a component doesn't exist or not found in the specified type")
		}
		kind, found, err := unstructured.NestedString(workload, "kind")
		if !found || err != nil {
			return nil, errors.New("workload kind in a component doesn't exist or not found in the specified type")
		}

		compMetadata, found, err := unstructured.NestedMap(comp, "metadata")
		if !found || err != nil {
			return nil, errors.New("component metadata doesn't exist or not found in the specified type")
		}
		name, found, err := unstructured.NestedString(compMetadata, "name")
		if !found || err != nil {
			return nil, errors.New("component name doesn't exist or not found in the specified type")
		}

		//Checking if the specific component name is present in the component names array
		//where component names array is the array of component names
		//which has ingress traits applied on it

		for i := range conversionComponents {
			if conversionComponents[i].ComponentName == name {

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

					conversionComponents[i].Weblogicworkload = weblogicMap

				case "VerrazzanoHelidonWorkload":
					//Appending the helidon workloads in the helidon workload array
					helidonWorkload := &unstructured.Unstructured{}
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
					coherenceWorkload := &unstructured.Unstructured{}
					workloadJSON, err := json.Marshal(workload)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(workloadJSON, &coherenceWorkload)
					if err != nil {
						fmt.Printf("Failed to unmarshal: %v\n", err)

					}
					conversionComponents[i].Coherenceworkload = coherenceWorkload
				case "Service":
					service := &corev1.Service{}
					workloadJSON, err := json.Marshal(workload)
					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(workloadJSON, &service)
					if err != nil {
						fmt.Printf("Failed to unmarshal: %v\n", err)

					}
					conversionComponents[i].Service = service
				}

			}
		}
	}
	return conversionComponents, nil

}
