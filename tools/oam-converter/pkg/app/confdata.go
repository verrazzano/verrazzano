// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app

import (
	"encoding/json"
	"errors"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
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
	var outputDirectory string
	inputDirectory = os.Args[1]
	outputDirectory = os.Args[2]

	var appData []map[string]interface{}

	var components []map[string]interface{}

	//iterate through user inputted directory
	files, _ := iterateDirectory(inputDirectory)
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
				errors.New("kind was not found as a string field")
			}
			compApiVersion, found, err := unstructured.NestedString(component, "apiVersion")
			if !found || err != nil {
				errors.New("apiVersion was not found as a string field")
			}
			if compKind == "Component" && compApiVersion == "core.oam.dev/v1alpha2" {
				components = append(components, component)
			}
			if compKind == "ApplicationConfiguration" && compApiVersion == "core.oam.dev/v1alpha2" {
				appData = append(appData, component)
			}
			//TODO: If Kind is neither Component or AppConfig, return YAML

		}
	}

	conversionComponents, err := traits.ExtractTrait(appData)

	if err != nil {
		return err
	}
	conversionComponents, err = traits.ExtractWorkload(components, conversionComponents)
	if err != nil {
		return err
	}

	outputResources, err := resources.CreateKubeResources(conversionComponents)

	writeKubeResources(outputDirectory, outputResources)

	if err != nil {
		return err
	}
	return nil
}

func iterateDirectory(path string) ([]string, error){
	var files []string

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(info.Name(), "yaml") || strings.Contains(info.Name(), "yml") {
			files = append(files, path)
		}
		return nil
	})
	return files, nil
}
func writeKubeResources(outputDirectory string, outputResources *types.KubeRecources)(error){
	fileName := "output.yaml"
	filePath := filepath.Join(outputDirectory, fileName)
	f, err := os.Create(filePath)
	var output string
	if err != nil {
		return err
	}
	if outputResources.VirtualServices != nil {
		for _, virtualservice := range outputResources.VirtualServices {
			r, err := json.Marshal(virtualservice)
			if err != nil {
				return err
			}
			out, err := yaml.JSONToYAML(r)
			if err != nil {
				return err
			}
			output = output + "---\n" + string(out)
		}
	}
	if outputResources.Gateways != nil {
		for _, gateway := range outputResources.Gateways {
			r, err := json.Marshal(gateway)
			if err != nil {
				return err
			}
			out, err := yaml.JSONToYAML(r)
			if err != nil {
				return err
			}
			output = output + "---\n" + string(out)
		}
	}
	if outputResources.DestinationRules != nil {
		for _, destinationrule := range outputResources.DestinationRules {
			r, err := json.Marshal(destinationrule)
			if err != nil {
				return err
			}
			out, err := yaml.JSONToYAML(r)
			if err != nil {
				return err
			}
			output = output + "---\n" + string(out)
		}
	}
	if outputResources.AuthPolicies != nil {
		for _, authpolicy := range outputResources.AuthPolicies {
			r, err := json.Marshal(authpolicy)
			if err != nil {
				return err
			}
			out, err := yaml.JSONToYAML(r)
			if err != nil {
				return err
			}
			output = output + "---\n" + string(out)
		}
	}
	if outputResources.ServiceMonitors != nil {
		for _, servicemonitor := range outputResources.ServiceMonitors {
			r, err := json.Marshal(servicemonitor)
			if err != nil {
				return err
			}
			out, err := yaml.JSONToYAML(r)
			if err != nil {
				return err
			}
			output = output + "---\n" + string(out)
		}
	}
	defer f.Close()

	_, err2 := f.WriteString(string(output))
	if err2 != nil {
		log.Fatal(err2)
	}
	return nil
}