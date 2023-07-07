package coherenceresources

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	"google.golang.org/protobuf/types/known/durationpb"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"time"
)

// createOfUpdateDestinationRule creates or updates the DestinationRule.
func createDestinationRuleFromCoherenceWorkload(trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, coherenceWorkload *vzapi.VerrazzanoCoherenceWorkload) error {
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
		return mutateDestinationRuleFromCoherenceWorkload(destinationRule, rule, namespace, coherenceWorkload)

	}
	return nil
}

// mutateDestinationRule changes the destination rule based upon a traits configuration
func mutateDestinationRuleFromCoherenceWorkload(destinationRule *istioclient.DestinationRule, rule vzapi.IngressRule, namespace *corev1.Namespace, coherenceWorkload *vzapi.VerrazzanoCoherenceWorkload) error {
	dest, err := createDestinationFromRuleOrCoherenceWorkload(rule, coherenceWorkload)
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
	return nil
}
