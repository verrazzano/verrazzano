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
	workload "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/workloads"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

func ConfData() error {

	var inputDirectory string
	var outputDirectory string

	//Check the length of args
	if len(os.Args) != 3 {
		return fmt.Errorf("Not enough args to run tool")
	}
	inputDirectory = os.Args[1]
	outputDirectory = os.Args[2]

	//used to store app file data
	var appData []map[string]interface{}

	//used to store comp file data
	var components []map[string]interface{}

	//used to store non-oam file data
	var k8sResources []map[string]interface{}

	//iterate through user inputted directory
	files, err := iterateDirectory(inputDirectory)
	if err != nil {
		return fmt.Errorf("error in iterating over directory %w", err)
	}

	//Read each file from the input directory
	for _, input := range files {
		data, _ := ioutil.ReadFile(input)
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
			compVersion, found, err := unstructured.NestedString(component, "apiVersion")
			if !found || err != nil {
				return errors.New("component api version doesn't exist or not found in the specified type")
			}
			//Check the kind of each component and apiVersion
			if compKind == "Component" && compVersion == consts.CompAPIVersion {
				components = append(components, component)
			} else if compKind == "ApplicationConfiguration" && compVersion == consts.CompAPIVersion {
				appData = append(appData, component)
			} else {
				//For generic workload
				//TODO for K8s resources that are not OAM-specific
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
	conversionComponents, err = workload.ExtractWorkload(components, conversionComponents)
	if err != nil {
		return errors.New("error in extracting workload")
	}

	//Create child resources
	outputResources, err := resources.CreateResources(conversionComponents)
	if err != nil {
		return err
	}
	//Write the K8s child resources to the file
	err = writeKubeResources(outputDirectory, outputResources)
	if err != nil {
		return err
	}
	return nil
}

// Write the kube resources to the files in output directory
func writeKubeResources(outputDirectory string, outputResources *types.KubeResources) (err error) {

	//Write virtual services to files
	if outputResources.VirtualServices != nil {
		for _, virtualService := range outputResources.VirtualServices {

			fileName := "" + virtualService.Name + ".yaml"
			filePath := filepath.Join(outputDirectory, fileName)

			f, err := os.Create(filePath)
			if err != nil {
				return err
			}
			r, err := json.Marshal(virtualService)
			if err != nil {
				return err
			}
			output, err := yaml.JSONToYAML(r)
			if err != nil {
				return err
			}

			// Use defer with an anonymous function to close the file after it's processed
			defer func() {
				if err := f.Close(); err != nil {
					err = fmt.Errorf("failed to close file: %w", err)
				}
			}()

			_, err = f.WriteString(string(output))
			if err != nil {
				return err
			}

		}
	}

	//Write down gateway to files
	if outputResources.Gateway != nil {

		fileName := "gateway.yaml"
		filePath := filepath.Join(outputDirectory, fileName)

		f, err := os.Create(filePath)
		if err != nil {
			return err
		}

		gatewayYaml, err := yaml.Marshal(outputResources.Gateway)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}

		_, err = f.WriteString(string(gatewayYaml))
		if err != nil {
			return fmt.Errorf("failed to write YAML to file: %w", err)
		}
		// Use defer with an anonymous function to close the file after it's processed
		defer func() {
			if err := f.Close(); err != nil {
				err = fmt.Errorf("failed to close file: %w", err)
			}
		}()
	}

	//Write down destination rules to files
	if outputResources.DestinationRules != nil {
		for _, destinationRule := range outputResources.DestinationRules {

			if destinationRule != nil {
				fileName := "" + destinationRule.Name
				filePath := filepath.Join(outputDirectory, fileName)

				f, err := os.Create(filePath)
				if err != nil {
					return err
				}
				r, err := json.Marshal(destinationRule)
				if err != nil {
					return err
				}
				output, err := yaml.JSONToYAML(r)
				if err != nil {
					return err
				}

				// Use defer with an anonymous function to close the file after it's processed
				defer func() {
					if err := f.Close(); err != nil {
						err = fmt.Errorf("failed to close file: %w", err)
					}
				}()

				_, err2 := f.WriteString(string(output))
				if err2 != nil {
					return err2
				}
			}
		}
	}

	//Write down Authorization Policies to files
	if outputResources.AuthPolicies != nil {
		for _, authPolicy := range outputResources.AuthPolicies {

			if authPolicy != nil {
				fileName := "authzpolicy.yaml"
				filePath := filepath.Join(outputDirectory, fileName)

				f, err := os.Create(filePath)
				if err != nil {
					return err
				}
				r, err := json.Marshal(authPolicy)
				if err != nil {
					return err
				}
				output, err := yaml.JSONToYAML(r)
				if err != nil {
					return err
				}

				// Use defer with an anonymous function to close the file after it's processed
				defer func() {
					if err := f.Close(); err != nil {
						err = fmt.Errorf("failed to close file: %w", err)
					}
				}()

				_, err2 := f.WriteString(string(output))
				if err2 != nil {
					return err2
				}

			}

		}
	}

	//Write down Service Monitors to files
	if outputResources.ServiceMonitors != nil {
		var output string
		fileName := "output.yaml"
		filePath := filepath.Join(outputDirectory, fileName)
		f, err := os.Create(filePath)
		if err != nil {
			return err
		}
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

		defer f.Close()

		_, err2 := f.WriteString(string(output))
		if err2 != nil {
			return err2
		}

	}
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
