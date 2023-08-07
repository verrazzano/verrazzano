// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package app

import (
	"encoding/json"
	"errors"
	"fmt"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/traits"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

func ConfData() error {
	//used to store app file data
	var appData []map[string]interface{}

	//used to store comp file data
	var components []map[string]interface{}

	//used to store non-oam file data
	var k8sResources []any

	//iterate through user inputted directory
	files, err := iterateDirectory(types.InputArgs.InputDirectory)
	if err != nil {
		fmt.Println("Error in iterating over directory", err)
	}
	//Read each file from the input directory
	for _, input := range files {
		data, err := ioutil.ReadFile(input)
		if err != nil {
			return errors.New("error in unmarshalling the components")
		}
		datastr := string(data)
		//Split the objects using "---" delimiter
		objects := strings.Split(datastr, "---")

		//Iterate over each object to segregate into applicationConfiguration kind or Component kind
		for _, obj := range objects {
			var component map[string]interface{}
			err := yaml.Unmarshal([]byte(obj), &component)
			if err != nil {
				return errors.New("error in unmarshalling the components")
			}
			compKind, found, err := unstructured.NestedString(component, "kind")
			if !found || err != nil {
				return errors.New("component kind doesn't exist or not found in the specified type")
			}
			compAPIVersion, found, err := unstructured.NestedString(component, "apiVersion")
			if !found || err != nil {
				return errors.New("component api version doesn't exist or not found in the specified type")
			}
			//Check the kind od each component and apiVersion
			if compKind == "Component" && compAPIVersion == consts.CompAPIVersion {
				components = append(components, component)
			} else if compKind == "ApplicationConfiguration" && compAPIVersion == consts.CompAPIVersion {
				appData = append(appData, component)
			} else {
				k8sResources = append(k8sResources, component)
			}
		}
	}

	//Extract traits from app file
	conversionComponents, err := traits.ExtractTrait(appData)

	if err != nil {
		return errors.New("failed extracting traits from app")
	}
	//Extract workloads from app file
	conversionComponents, err = traits.ExtractWorkload(components, conversionComponents)
	if err != nil {
		return errors.New("error in extracting workload")
	}

	//Create child resources
	outputResources, err := resources.CreateResources(conversionComponents)
	if err != nil {
		return err
	}
	var byteResources []unstructured.Unstructured

	for _, obj := range outputResources {
		resource, err := ToUnstructured(obj)
		if err != nil {
			print("Null obj")
		}
		byteResources = append(byteResources, resource...)
	}
	//Write the K8s child resources to the file
	err = writeKubeResources(types.InputArgs.OutputDirectory, byteResources)
	if err != nil {
		return err
	}
	//write the nonOAM resources to the file
	return nil
}

// Write the kube resources to the files in output directory
func writeKubeResources(outputDirectory string, outputResources []unstructured.Unstructured) error {
	for index := range outputResources {
		err := writeToDirectory(outputDirectory, outputResources[index])
		if err != nil {
			return err
		}
	}
	return nil
}

func writeToDirectory(outputDirectory string, index unstructured.Unstructured) error {
	var fileName string
	var filePath string
	//check to find out what resource is being manipulated and printed
	switch index.GetKind() {
	case "Gateway":
		fileName = "gateway.yaml"
		filePath = filepath.Join(outputDirectory, fileName)
	case "VirtualService":
		fileName = "virtualservice.yaml"
		filePath = filepath.Join(outputDirectory, fileName)
	case "DestinationRule":
		fileName = "destinationrule.yaml"
		filePath = filepath.Join(outputDirectory, fileName)
	case "AuthorizationPolicy":
		fileName = "authorizationpolicy.yaml"
		filePath = filepath.Join(outputDirectory, fileName)
	case "ServiceMonitor":
		fileName = "servicemonitor.yaml"
		filePath = filepath.Join(outputDirectory, fileName)
	default:
		return fmt.Errorf("Unsupported datq type%v")
	}

	//print resources in respective files
	writeToFile(filePath, index)
	return nil
}
func ToUnstructured(o any) ([]unstructured.Unstructured, error) {
	j, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	obj, err := runtime.Decode(unstructured.UnstructuredJSONScheme, j)
	if err != nil {
		return nil, err
	}
	if u, ok := obj.(*unstructured.Unstructured); ok {
		return []unstructured.Unstructured{*u}, nil
	}
	if us, ok := obj.(*unstructured.UnstructuredList); ok {
		return us.Items, nil
	}

	return nil, errors.New("unknown object type during unstructured serialization")
}
func writeToFile(filePath string, object any) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	r, err := json.Marshal(object)
	if err != nil {
		return err
	}
	output, err := yaml.JSONToYAML(r)
	if err != nil {
		return err
	}
	_, err = f.WriteString(string(output))
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

// Iterate over input directory
func iterateDirectory(path string) ([]string, error) {
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
