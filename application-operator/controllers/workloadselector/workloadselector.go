// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloadselector

import (
	"context"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// WorkloadSelector type for accessing functions
type WorkloadSelector struct {
	KubeClient kubernetes.Interface
}

// DoesWorkloadMatch returns a boolean indicating whether an unstructured resource matches any of the criteria for a match.
// The criteria used to match is a namespace label selector, object label selector, and group, version,
// and kind of resource.
func (w *WorkloadSelector) DoesWorkloadMatch(workload *unstructured.Unstructured, namespaceSelector *metav1.LabelSelector, objectSelector *metav1.LabelSelector, apiGroups []string, apiVersions []string, resources []string) (bool, error) {
	// Check if we match the given namespace label selector
	found, err := w.doesNamespaceMatch(workload, namespaceSelector)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	// If the namespace matches then check if we match the given object label selector
	return w.doesObjectMatch(workload, objectSelector, apiGroups, apiVersions, resources)
}

// doesNamespaceMatch returns a boolean indicating whether an unstructured resource matches the namespace selector
func (w *WorkloadSelector) doesNamespaceMatch(workload *unstructured.Unstructured, namespaceSelector *metav1.LabelSelector) (bool, error) {
	// If the namespace label selector is not specified then we don't need to check the namespace
	if namespaceSelector == nil || reflect.DeepEqual(namespaceSelector, metav1.LabelSelector{}) {
		return true, nil
	}

	// Get the namespace object for the workload resource
	namespace, err := w.KubeClient.CoreV1().Namespaces().Get(context.TODO(), workload.GetNamespace(), metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	// Check if the namespace labels match the namespace label selector
	label, err := metav1.LabelSelectorAsSelector(namespaceSelector)
	if err != nil {
		return false, err
	}
	if label.Matches(labels.Set(namespace.GetLabels())) {
		return true, nil
	}

	return false, nil
}

// doesObjectMatch returns a boolean indicating whether an unstructured resource matches the criteria for a match.
// The criteria used to match is an object label selector, and group, version, and kind values
func (w *WorkloadSelector) doesObjectMatch(workload *unstructured.Unstructured, objectSelector *metav1.LabelSelector, apiGroups []string, apiVersions []string, resources []string) (bool, error) {
	// Get the group and version of the workload resource
	gv, err := schema.ParseGroupVersion(workload.GetAPIVersion())
	if err != nil {
		return false, nil
	}

	// Check that the workload resource GVK matches expected GVKs
	if !checkMatch(gv.Version, apiVersions) || !checkMatch(gv.Group, apiGroups) || !checkMatch(workload.GetKind(), resources) {
		return false, nil
	}

	// If the object label selector is not specified then we don't need to check the resource for a match
	if objectSelector == nil || reflect.DeepEqual(objectSelector, metav1.LabelSelector{}) {
		return true, nil
	}
	// Check if the workload resource labels match the object label selector
	label, err := metav1.LabelSelectorAsSelector(objectSelector)
	if err != nil {
		return false, err
	}
	if label.Matches(labels.Set(workload.GetLabels())) {
		return true, nil
	}

	return false, nil
}

// checkMatch checks for a matching string within a string array
func checkMatch(match string, matches []string) bool {
	if len(matches) == 0 {
		return true
	}
	for _, value := range matches {
		if value == "*" {
			return true
		}
		if value == strings.ToLower(match) {
			return true
		}
	}

	return false
}
