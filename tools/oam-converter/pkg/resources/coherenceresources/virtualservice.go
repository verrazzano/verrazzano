// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherenceresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationRule"
	vs "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/virtualservice"
	istio "istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// creates the VirtualService child resource of the trait.
func createVirtualServiceFromCoherenceWorkload(ingresstrait *vzapi.IngressTrait, rule vzapi.IngressRule,
	allHostsForTrait []string, name string, gateway *vsapi.Gateway, coherenceWorkload *vzapi.VerrazzanoCoherenceWorkload) error {
	virtualService := &vsapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: consts.VirtualServiceAPIVersion,
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ingresstrait.Namespace,
			Name:      name,
		},
	}
	return mutateVirtualServiceFromCoherenceWorkload(virtualService, rule, allHostsForTrait, gateway, coherenceWorkload)
}

// mutateVirtualService mutates the output virtual service resource
func mutateVirtualServiceFromCoherenceWorkload(virtualService *vsapi.VirtualService, rule vzapi.IngressRule, allHostsForTrait []string, gateway *vsapi.Gateway, coherenceWorkload *vzapi.VerrazzanoCoherenceWorkload) error {
	virtualService.Spec.Gateways = []string{gateway.Name}
	virtualService.Spec.Hosts = allHostsForTrait
	matches := []*istio.HTTPMatchRequest{}
	paths := vs.GetPathsFromRule(rule)
	for _, path := range paths {
		matches = append(matches, &istio.HTTPMatchRequest{
			Uri: vs.CreateVirtualServiceMatchURIFromIngressTraitPath(path)})
	}
	dest, err := destination.CreateDestinationFromRule(rule)
	if err != nil {
		print(err)
		return err
	}
	route := istio.HTTPRoute{
		Match: matches,
		Route: []*istio.HTTPRouteDestination{dest}}
	virtualService.Spec.Http = []*istio.HTTPRoute{&route}

	fmt.Println("virtual-service", virtualService)
	directoryPath := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/"
	fileName := "vs.yaml"
	filePath := filepath.Join(directoryPath, fileName)

	virtualServiceYaml, err := yaml.Marshal(virtualService)
	if err != nil {
		fmt.Printf("Failed to marshal: %v\n", err)
		return err
	}
	// Write the YAML content to the file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return err
	}
	defer file.Close()

	// Append the YAML content to the file
	_, err = file.Write(virtualServiceYaml)
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	_, err = file.WriteString("---\n")
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	return nil
}
