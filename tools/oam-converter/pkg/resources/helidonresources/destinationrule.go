// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	destination "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/destinationRule"

	"google.golang.org/protobuf/types/known/durationpb"
	istionet "istio.io/api/networking/v1alpha3"
	istio "istio.io/api/networking/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"time"
)

// createOfUpdateDestinationRule creates or updates the DestinationRule.
func createDestinationRuleFromHelidonWorkload(trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, helidonWorkload *unstructured.Unstructured) error {
	if rule.Destination.HTTPCookie != nil {
		destinationRule := &istioclient.DestinationRule{
			TypeMeta: metav1.TypeMeta{
				APIVersion: consts.DestinationRuleAPIVersion,
				Kind:       "DestinationRule"},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: trait.Namespace,
				Name:      name},
		}
		namespace := &corev1.Namespace{}
		fmt.Println("destinationRule", destinationRule)
		return mutateDestinationRuleFromHelidonWorkload(destinationRule, rule, namespace, helidonWorkload)

	}
	return nil
}

// mutateDestinationRule changes the destination rule based upon a traits configuration
func mutateDestinationRuleFromHelidonWorkload(destinationRule *istioclient.DestinationRule, rule vzapi.IngressRule, namespace *corev1.Namespace, helidonWorkload *unstructured.Unstructured) error {
	dest, err := createDestinationFromRuleOrHelidonWorkload(rule, helidonWorkload)
	if err != nil {
		print(err)
		return err
	}

	mode := istionet.ClientTLSSettings_DISABLE
	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		mode = istionet.ClientTLSSettings_ISTIO_MUTUAL
	}
	destinationRule.Spec = istionet.DestinationRule{
		Host: dest.Destination.Host,
		TrafficPolicy: &istionet.TrafficPolicy{
			Tls: &istionet.ClientTLSSettings{
				Mode: mode,
			},
			LoadBalancer: &istionet.LoadBalancerSettings{
				LbPolicy: &istionet.LoadBalancerSettings_ConsistentHash{
					ConsistentHash: &istionet.LoadBalancerSettings_ConsistentHashLB{
						HashKey: &istionet.LoadBalancerSettings_ConsistentHashLB_HttpCookie{
							HttpCookie: &istionet.LoadBalancerSettings_ConsistentHashLB_HTTPCookie{
								Name: rule.Destination.HTTPCookie.Name,
								Path: rule.Destination.HTTPCookie.Path,
								Ttl:  durationpb.New(rule.Destination.HTTPCookie.TTL * time.Second)},
						},
					},
				},
			},
		},
	}
	fmt.Println("DestinationRule", destinationRule)
	directoryPath := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/"
	fileName := "dr.yaml"
	filePath := filepath.Join(directoryPath, fileName)

	destinationRuleYaml, err := yaml.Marshal(destinationRule)
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
	_, err = file.Write(destinationRuleYaml)
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

// createDestinationFromRuleOrService creates a destination from either the rule or the service.
// If the rule contains destination information that is used.
func createDestinationFromRuleOrHelidonWorkload(rule vzapi.IngressRule, helidonWorkload *unstructured.Unstructured) (*istio.HTTPRouteDestination, error) {
	if len(rule.Destination.Host) > 0 {

		dest, err := destination.CreateDestinationFromRule(rule)
		if err != nil {
			return nil, err
		}
		return dest, err

	}

	return createDestinationFromHelidonWorkload(rule.Destination.Port, helidonWorkload)

}

func createDestinationFromHelidonWorkload(rulePort uint32, helidonWorkload *unstructured.Unstructured) (*istio.HTTPRouteDestination, error) {
	//	Spec.DeploymentTemplate.PodSpec.Containers[0].Ports[0].ContainerPort != 0 {
	//	dest := istio.HTTPRouteDestination{
	//		Destination: &istio.Destination{Host: helidonWorkload.Name}}
	//	// Set the port to rule destination port
	//	dest.Destination.Port = &istio.PortSelector{Number: uint32(helidonWorkload.Spec.DeploymentTemplate.PodSpec.Containers[0].Ports[0].ContainerPort)}
	//	return &dest, nil
	//}
	return nil, fmt.Errorf("unable to select service for specified destination port %d", rulePort)
}
