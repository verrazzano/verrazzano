// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	"io/ioutil"
	"log"
	"sigs.k8s.io/yaml"
	"strings"
)

func ConfData() error {

	//Read app File
	appData, err := ioutil.ReadFile("/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/pkg/app/hello-helidon-app.yaml")
	if err != nil {
		fmt.Println("Failed to read YAML file:", err)
		return err
	}

	//Read Comp file
	compData, err := ioutil.ReadFile("/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/pkg/app/hello-helidon-comp.yaml")
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
	conversionComponents, err := traits.ExtractTrait(appMap)
	if err != nil {
		return errors.New("failed extracting traits from app")
	}
	conversionComponents, err = traits.ExtractWorkload(components, conversionComponents)
	if err != nil {
		return errors.New("error in extractingthe trait and workload - %s")
	}

	err = resources.CreateResources(conversionComponents)
	if err != nil {
		return err
	}
	return nil
}
