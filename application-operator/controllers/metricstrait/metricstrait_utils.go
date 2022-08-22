// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	vznav "github.com/verrazzano/verrazzano/application-operator/controllers/navigation"
	k8score "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// updateStringMap updates a string key value pair in a map.
// strMap is the map to be updated.  It may be nil.
// key is the key to add to the map
// value is the value to add to the map
// Returns the provided or a new map if strMap was nil
func updateStringMap(strMap map[string]string, key string, value string) map[string]string {
	if strMap == nil {
		strMap = map[string]string{}
	}
	strMap[key] = value
	return strMap
}

// copyStringMapEntries copies key value pairs from one map to another.
// target is the map key value pairs are copied into
// source is the map key value pairs are copied from
// keys are a list of keys to copy from the source to the target map
// Returns the target map or a new map if the target was nil
func copyStringMapEntries(target map[string]string, source map[string]string, keys ...string) map[string]string {
	if target == nil {
		target = map[string]string{}
	}
	for _, key := range keys {
		value, found := source[key]
		if found {
			target[key] = value
		}
	}
	return target
}

// parseYAMLString parses a string into a internal representation.
// s is the YAML formatted string to parse.
// Returns an unstructured representation of the input YAML string.
// Returns and error if parsing fails.
func parseYAMLString(s string) (*gabs.Container, error) {
	prometheusJSON, _ := yaml.YAMLToJSON([]byte(s))
	return gabs.ParseJSON(prometheusJSON)
}

// writeYAMLString writes unstructured data to a YAML formatted string.
// c is the unstructured representation
// Returns a YAML format string version of the input
// Returns an error if the unstructured cannot be converted to a YAML string.
func writeYAMLString(c *gabs.Container) (string, error) {
	bytes, err := yaml.JSONToYAML(c.Bytes())
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// getClusterNameFromObjectMetaOrDefault extracts the customer name from object metadata.
// meta is the object metadata to extract the cluster from.
// Returns the cluster name or "default" if the cluster name is the empty string.
func getClusterNameFromObjectMetaOrDefault(_ metav1.ObjectMeta) string {
	// Used to return ObjectName.ClusterName, however that field is obsolete and being removed from the k8s API
	return "default"
}

// getNamespaceFromObjectMetaOrDefault extracts the namespace name from object metadata.
// meta is the object metadata to extract the namespace name from.
// Returns the namespace name of "default" if the namespace is the empty string.
func getNamespaceFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.Namespace
	if name == "" {
		return "default"
	}
	return name
}

// mergeTemplateWithContext merges a map of string into a string template.
// template is the string to merge the values from the context map into.
// context is a map of string to be merged into the template.
// Returns a string with all values from the context merged into the template.
func mergeTemplateWithContext(template string, context map[string]string) string {
	for key, value := range context {
		template = strings.ReplaceAll(template, key, value)
	}
	return template
}

// GetSupportedWorkloadType returns workload type corresponding to input API version and kind
// that is supported by MetricsTrait.
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

// createServiceMonitorName creates a Prometheus scrape configmap job name from a trait.
// Format is {oam_app}_{cluster}_{namespace}_{oam_comp}
func createServiceMonitorName(trait *vzapi.MetricsTrait, portNum int) (string, error) {
	cluster := getClusterNameFromObjectMetaOrDefault(trait.ObjectMeta)
	namespace := getNamespaceFromObjectMetaOrDefault(trait.ObjectMeta)
	app, found := trait.Labels[appObjectMetaLabel]
	if !found {
		return "", fmt.Errorf("metrics trait missing application name label")
	}
	comp, found := trait.Labels[compObjectMetaLabel]
	if !found {
		return "", fmt.Errorf("metrics trait missing component name label")
	}
	portStr := ""
	if portNum > 0 {
		portStr = fmt.Sprintf("_%d", portNum)
	}

	finalName := fmt.Sprintf("%s_%s_%s_%s%s", app, cluster, namespace, comp, portStr)
	// Check for Kubernetes name length requirement
	if len(finalName) > 63 {
		finalName = fmt.Sprintf("%s_%s%s", app, namespace, portStr)
		if len(finalName) > 63 {
			return finalName[:63], nil
		}
	}
	return finalName, nil
}

// getPortSpecs returns a complete set of port specs from the trait and the trait defaults
func getPortSpecs(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec) []vzapi.PortSpec {
	ports := trait.Spec.Ports
	if len(ports) == 0 {
		// create a port spec from the existing port
		ports = []vzapi.PortSpec{{Port: trait.Spec.Port, Path: trait.Spec.Path}}
	} else {
		// if there are existing ports and a port/path setting, add the latter to the ports
		if trait.Spec.Port != nil {
			// add the port to the ports
			path := trait.Spec.Path
			if path == nil {
				path = traitDefaults.Path
			}
			portSpec := vzapi.PortSpec{
				Port: trait.Spec.Port,
				Path: path,
			}
			ports = append(ports, portSpec)
		}
	}
	return ports
}

func isEnabled(trait *vzapi.MetricsTrait) bool {
	return trait.Spec.Enabled == nil || *trait.Spec.Enabled
}

// useHTTPSForScrapeTarget returns true if https with Istio certs should be used for scrape target. Otherwise return false, use http
func useHTTPSForScrapeTarget(ctx context.Context, c client.Client, trait *vzapi.MetricsTrait) (bool, error) {
	if trait.Spec.WorkloadReference.Kind == "VerrazzanoCoherenceWorkload" || trait.Spec.WorkloadReference.Kind == "Coherence" {
		return false, nil
	}
	// Get the namespace resource that the MetricsTrait is deployed to
	namespace := &k8score.Namespace{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: "", Name: trait.Namespace}, namespace); err != nil {
		return false, err
	}
	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		return true, nil
	}
	return false, nil
}

// fetchSourceCredentialsSecretIfRequired fetches the metrics endpoint authentication credentials if a secret is provided.
func fetchSourceCredentialsSecretIfRequired(ctx context.Context, trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, workload *unstructured.Unstructured, cli client.Client) (*k8score.Secret, error) {
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
	secretNamespace, found, err := unstructured.NestedString(workload.Object, "metadata", "namespace")
	if err != nil {
		return nil, fmt.Errorf("failed to determine namespace for secret %s: %w", *secretName, err)
	}
	if !found {
		return nil, fmt.Errorf("failed to find namespace for secret %s", *secretName)
	}
	// Fetch the secret.
	secretKey := client.ObjectKey{Namespace: secretNamespace, Name: *secretName}
	secretObj := k8score.Secret{}
	err = cli.Get(ctx, secretKey, &secretObj)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %v: %w", secretKey, err)
	}
	return &secretObj, nil
}

// isWLSWorkload returns true if the unstructured object is a Weblogic Workload
func isWLSWorkload(workload *unstructured.Unstructured) (bool, error) {
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
