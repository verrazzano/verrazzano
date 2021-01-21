// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"github.com/Jeffail/gabs/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
	"strings"
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

// stringSliceContainsString determines if a string is found in a string slice.
// slice is the string slice to search. May be nil.
// find is the string to search for in the slice.
// Returns true if the string is found in the slice and false otherwise.
func stringSliceContainsString(slice []string, find string) bool {
	for _, s := range slice {
		if s == find {
			return true
		}
	}
	return false
}

// removeStringFromStringSlice removes a string from a string slice.
// slice is the string slice to remove the string from. May be nil.
// remove is the string to remove from the slice.
// Returns a new slice with the remove string removed.
func removeStringFromStringSlice(slice []string, remove string) []string {
	result := []string{}
	for _, s := range slice {
		if s == remove {
			continue
		}
		result = append(result, s)
	}
	return result
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
func getClusterNameFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.ClusterName
	if name == "" {
		return "default"
	}
	return name
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
