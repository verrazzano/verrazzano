// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app

import (
	"errors"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

func ConfData() error {

	var inputDirectory string
	//var outputDirectory string
	inputDirectory = os.Args[1]
	//outputDirectory = os.Args[2]
	var appData []map[string]interface{}

	var components []map[string]interface{}

	//iterate through user inputted directory
	files := iterateDirectory(inputDirectory)
	var data []byte
	//Loop through all app files and store into appDataArr
	for _, input := range files {
		data, _ = ioutil.ReadFile(input)
		datastr := string(data)
		objects := strings.Split(datastr, "---")

		for _, obj := range objects {
			var component map[string]interface{}
			err := yaml.Unmarshal([]byte(obj), &component)
			if err != nil {
				log.Fatalf("Failed to unmarshal YAML: %v", err)
			}
			compKind, found, err := unstructured.NestedString(component, "kind")
			if !found || err != nil {
				return err
			}
			compApiVersion, found, err := unstructured.NestedString(component, "apiVersion")
			if !found || err != nil {
				return err
			}
			if compKind == "Component" && compApiVersion == "core.oam.dev/v1alpha2" {
				components = append(components, component)
			}
			if compKind == "ApplicationConfiguration" && compApiVersion == "core.oam.dev/v1alpha2" {
				appData = append(appData, component)
			}

		}
	}

	conversionComponents, err := traits.ExtractTrait(appData)

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

func iterateDirectory(path string) []string {
	var files []string

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatalf(err.Error())
		}
		if strings.Contains(info.Name(), "yaml") || strings.Contains(info.Name(), "yml") {
			files = append(files, path)
		}
		return nil
	})
	return files
}
