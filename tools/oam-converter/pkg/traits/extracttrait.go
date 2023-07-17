// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package traits

import (
	"encoding/json"
	"errors"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	coherence "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/coherenceresources"
	weblogic "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/resources/weblogicresources"
	"github.com/verrazzano/verrazzano/tools/oam-converter/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"log"
	"reflect"
	"regexp"
	"strings"
)

// ExtractTrait - Extract traits from the app map
func ExtractTrait(appMaps []map[string]interface{}) ([]*types.ConversionComponents, error) {
	conversionComponents := []*types.ConversionComponents{}
	for _, appMap := range appMaps {
		appMetadata, found, err := unstructured.NestedMap(appMap, "metadata")
		if !found || err != nil {
			return nil, errors.New("app metadata doesn't exist")
		}
		appName, found, err := unstructured.NestedString(appMetadata, "name")
		if !found || err != nil {
			return nil, errors.New("app name key doesn't exist")
		}

		appNamespace, found, err := unstructured.NestedString(appMetadata, "namespace")
		if !found || err != nil {
			return nil, errors.New("namespace key doesn't exist")
		}

		appSpec, found, err := unstructured.NestedMap(appMap, "spec")
		if !found || err != nil {
			return nil, errors.New("app spec doesn't exist")
		}

		appComponents, found, err := unstructured.NestedSlice(appSpec, "components")
		if !found || err != nil {
			return nil, errors.New("app components doesn't exist")
		}

		for _, component := range appComponents {
			componentMap := component.(map[string]interface{})
			componentTraits, ok := componentMap[consts.YamlTraits].([]interface{})
			if ok && len(componentTraits) > 0 {
				for _, trait := range componentTraits {
					traitMap := trait.(map[string]interface{})
					//traitSpec := traitMap[consts.TraitComponent].(map[string]interface{})
					traitSpec, found, err := unstructured.NestedMap(traitMap, "trait")
					if !found || err != nil {
						return nil, errors.New("trait spec doesn't exist")

					}

					traitKind, found, err := unstructured.NestedString(traitSpec, "kind")
					if !found || err != nil {
						return nil, errors.New("trait kind doesn't exist")
					}
					if traitKind == consts.IngressTrait {
						ingressTrait := &vzapi.IngressTrait{}
						traitJSON, err := json.Marshal(traitSpec)

						if err != nil {
							fmt.Printf("Failed to marshal trait: %v", err)
						}

						err = json.Unmarshal(traitJSON, ingressTrait)

						if err != nil {
							fmt.Printf("Failed to unmarshal trait: %v", err)
						}

						conversionComponents = append(conversionComponents, &types.ConversionComponents{
							AppNamespace:  appNamespace,
							AppName:       appName,
							ComponentName: componentMap["componentName"].(string),
							IngressTrait:  ingressTrait,
						})
					}
					if traitKind == consts.MetricsTrait {
						metricsTrait := &vzapi.MetricsTrait{}
						traitJSON, err := json.Marshal(traitSpec)

						if err != nil {
							fmt.Printf("Failed to marshal trait: %v", err)
						}

						err = json.Unmarshal(traitJSON, metricsTrait)

						if err != nil {
							fmt.Printf("Failed to unmarshal trait: %v", err)
						}

						conversionComponents = append(conversionComponents, &types.ConversionComponents{
							AppNamespace:  appNamespace,
							AppName:       appName,
							ComponentName: componentMap["componentName"].(string),
							MetricsTrait:  metricsTrait,
						})
					}
				}
			}
		}
	}

	return conversionComponents, nil
}

// ExtractWorkload - Extract workload from comp map
func ExtractWorkload(components []map[string]interface{}, conversionComponents []*types.ConversionComponents) ([]*types.ConversionComponents, error) {
	weblogicMap := make(map[string]*vzapi.VerrazzanoWebLogicWorkload)
	for _, comp := range components {

		spec, found, err := unstructured.NestedMap(comp, "spec")
		if !found || err != nil {
			return nil, errors.New("spec key in a component doesn't exist")
		}
		workload, found, err := unstructured.NestedMap(spec, "workload")
		if !found || err != nil {
			return nil, errors.New("workload in a component doesn't exist")
		}
		kind, found, err := unstructured.NestedString(workload, "kind")
		if !found || err != nil {
			return nil, errors.New("workload kind in a component doesn't exist")
		}
		//name = comp["metadata"].(map[string]interface{})["name"].(string)
		compMetadata, found, err := unstructured.NestedMap(comp, "metadata")
		if !found || err != nil {
			return nil, errors.New("component metadata doesn't exist")
		}
		name, found, err := unstructured.NestedString(compMetadata, "name")
		if !found || err != nil {
			return nil, errors.New("component name doesn't exist")
		}

		//Checking if the specific component name is present in the component names array
		//where component names array is the array of component names
		//which has ingress traits applied on it

		for i := range conversionComponents {
			if conversionComponents[i].ComponentName == name {
				// Assign the desired value to the specific element
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

					//putting into map of workloads whose key is the component name and
					//value is the weblogic workload
					weblogicMap[name] = weblogicWorkload
					conversionComponents[i].WeblogicworkloadMap = weblogicMap

				case "VerrazzanoHelidonWorkload":
					//Appending the helidon workloads in the helidon workload array
					helidonWorkload := &vzapi.VerrazzanoHelidonWorkload{}
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
					coherenceWorkload := &vzapi.VerrazzanoCoherenceWorkload{}
					workloadJSON, err := json.Marshal(workload)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(workloadJSON, &coherenceWorkload)
					if err != nil {
						fmt.Printf("Failed to unmarshal: %v\n", err)

					}
					conversionComponents[i].Coherenceworkload = coherenceWorkload
				}
				break
			}
		}
	}
	return conversionComponents, nil

}
func FetchWorkloadFromTrait(trait *vzapi.MetricsTrait) (*unstructured.Unstructured, error) {
	var workload = &unstructured.Unstructured{}
	workload.SetAPIVersion(trait.GetWorkloadReference().APIVersion)
	workload.SetKind(trait.GetWorkloadReference().Kind)
	//workloadKey := client.ObjectKey{Name: trait.GetWorkloadReference().Name, Namespace: trait.GetNamespace()}

	return FetchWorkloadResource(workload)
}
func FetchContainedWorkload(workload *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	apiVersion, kind, _, err := GetContainedWorkloadVersionKindName(workload)
	if err != nil {
		return nil, err
	}
	_ = ""
	u := &unstructured.Unstructured{}
	u.SetAPIVersion(apiVersion)
	u.SetKind(kind)

	return u, nil
}
func FetchWorkloadResource(workload *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	// Getting kind of helidon workload i.e. "VerrazzanoHelidonWorkload"
	helidonWorkloadKind := reflect.TypeOf(vzapi.VerrazzanoHelidonWorkload{}).Name()
	// If the workload does not wrap unstructured data
	if !IsVerrazzanoWorkloadKind(workload) || (helidonWorkloadKind == workload.GetKind()) {
		return workload, nil
	}

	// this is one of our wrapper workloads so we need to unwrap and pull out the real workload
	resource, err := FetchContainedWorkload(workload)
	if err != nil {
		return nil, err
	}

	return resource, nil
}
func IsVerrazzanoWorkloadKind(workload *unstructured.Unstructured) bool {
	kind := workload.GetKind()
	return strings.HasPrefix(kind, "Verrazzano") && strings.HasSuffix(kind, "Workload")
}
func GetContainedWorkloadVersionKindName(workload *unstructured.Unstructured) (string, string, string, error) {
	gvk := WorkloadToContainedGVK(workload)
	if gvk == nil {
		return "", "", "", fmt.Errorf("unable to find contained GroupVersionKind for workload: %v", workload)
	}

	apiVersion, kind := gvk.ToAPIVersionAndKind()

	// NOTE: this may need to change if we do not allow the user to set the name or if we do and default it
	// to the workload or component name
	name, found, err := unstructured.NestedString(workload.Object, "spec", "template", "metadata", "name")
	if !found || err != nil {
		return "", "", "", fmt.Errorf("unable to find metadata name in contained workload")
	}

	return apiVersion, kind, name, nil
}
func WorkloadToContainedGVK(workload *unstructured.Unstructured) *schema.GroupVersionKind {
	if workload.GetKind() == vzconst.VerrazzanoWebLogicWorkloadKind {
		apiVersion, found, _ := unstructured.NestedString(workload.Object, "spec", "template", "apiVersion")
		var gvk schema.GroupVersionKind
		if found {
			gvk = schema.FromAPIVersionAndKind(apiVersion, "Domain")
		} else {
			gvk = schema.GroupVersionKind{Group: "weblogic.oracle", Version: "v8", Kind: "Domain"}
		}
		return &gvk
	}

	return APIVersionAndKindToContainedGVK(workload.GetAPIVersion(), workload.GetKind())
}
func APIVersionAndKindToContainedGVK(apiVersion string, kind string) *schema.GroupVersionKind {
	var workloadToContainedGVKMap = map[string]schema.GroupVersionKind{
		"oam.verrazzano.io/v1alpha1.VerrazzanoWebLogicWorkload":  {Group: "weblogic.oracle", Version: "v9", Kind: "Domain"},
		"oam.verrazzano.io/v1alpha1.VerrazzanoCoherenceWorkload": {Group: "coherence.oracle.com", Version: "v1", Kind: "Coherence"},
	}
	key := fmt.Sprintf("%s.%s", apiVersion, kind)
	gvk, ok := workloadToContainedGVKMap[key]
	if ok {
		return &gvk
	}
	return nil
}
func FetchTraitDefaults(workload *unstructured.Unstructured) (*vzapi.MetricsTraitSpec, bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		print(err)
	}

	workloadType := GetSupportedWorkloadType(apiVerKind)
	switch workloadType {
	case constants.WorkloadTypeWeblogic:
		spec, err := weblogic.NewTraitDefaultsForWLSDomainWorkload(workload)
		return spec, true, err
	case constants.WorkloadTypeCoherence:
		spec, err := coherence.NewTraitDefaultsForCOHWorkload(workload)
		return spec, true, err
	case constants.WorkloadTypeGeneric:
		spec, err := NewTraitDefaultsForGenericWorkload()
		return spec, true, err
	default:
		// Log the kind/workload is unsupported and return a nil trait.
		return nil, false, nil
	}

}
func GetSupportedWorkloadType(apiVerKind string) string {
	// Match any version of Group=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		return constants.WorkloadTypeWeblogic
	}
	// Match any version of Group=coherence.oracle and Kind=Coherence
	if matched, _ := regexp.MatchString("^coherence.oracle.com/.*\\.Coherence$", apiVerKind); matched {
		return constants.WorkloadTypeCoherence
	}

	// Match any version of Group=coherence.oracle and Kind=VerrazzanoHelidonWorkload or
	// In the case of Helidon, the workload isn't currently being unwrapped
	if matched, _ := regexp.MatchString("^oam.verrazzano.io/.*\\.VerrazzanoHelidonWorkload$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	// Match any version of Group=core.oam.dev and Kind=ContainerizedWorkload
	if matched, _ := regexp.MatchString("^core.oam.dev/.*\\.ContainerizedWorkload$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	// Match any version of Group=apps and Kind=Deployment
	if matched, _ := regexp.MatchString("^apps/.*\\.Deployment$", apiVerKind); matched {
		return constants.WorkloadTypeGeneric
	}

	return ""
}
func NewTraitDefaultsForGenericWorkload() (*vzapi.MetricsTraitSpec, error) {
	port := 8080
	path := "/metrics"
	return &vzapi.MetricsTraitSpec{
		Ports: []vzapi.PortSpec{{
			Port: &port,
			Path: &path,
		}},
		Path:   &path,
		Secret: nil,
		//Scraper: &r.Scraper
	}, nil
}
func FetchSourceCredentialsSecretIfRequired(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, workload *unstructured.Unstructured) (*corev1.Secret, error) {
	secretName := trait.Spec.Secret
	// If no secret name explicitly provided use the default secret name.
	if secretName == nil && traitDefaults != nil {
		secretName = traitDefaults.Secret
	}
	// If neither an explicit or default secret name provided do not fetch a secret.
	if secretName == nil {
		return nil, nil
	}
	// Use the workload namespace for the secret to fetch.
	//secretNamespace, found, err := unstructured.NestedString(workload.Object, "metadata", "namespace")
	//if err != nil {
	//	return nil, fmt.Errorf("failed to determine namespace for secret %s: %w", *secretName, err)
	//}
	//if !found {
	//	return nil, fmt.Errorf("failed to find namespace for secret %s", *secretName)
	//}
	//// Fetch the secret.
	//secretKey := client.ObjectKey{Namespace: secretNamespace, Name: *secretName}
	secretObj := corev1.Secret{}

	return &secretObj, nil
}
func UseHTTPSForScrapeTarget(trait *vzapi.MetricsTrait) (bool, error) {
	if trait.Spec.WorkloadReference.Kind == "VerrazzanoCoherenceWorkload" || trait.Spec.WorkloadReference.Kind == "Coherence" {
		return false, nil
	}
	// Get the namespace resource that the MetricsTrait is deployed to
	namespace := &corev1.Namespace{}

	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		return true, nil
	}
	return false, nil
}
func IsWLSWorkload(workload *unstructured.Unstructured) (bool, error) {
	apiVerKind, err := vznav.GetAPIVersionKindOfUnstructured(workload)
	if err != nil {
		return false, err
	}
	// Match any version of APIVersion=weblogic.oracle and Kind=Domain
	if matched, _ := regexp.MatchString("^weblogic.oracle/.*\\.Domain$", apiVerKind); matched {
		return true, nil
	}
	return false, nil
}