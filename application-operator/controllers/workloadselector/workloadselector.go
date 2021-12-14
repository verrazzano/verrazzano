// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloadselector

import (
	"context"
	"reflect"
	"strings"

	"github.com/gertd/go-pluralize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// WorkloadSelector type for accessing functions
type WorkloadSelector struct {
	KubeClient    kubernetes.Interface
	DynamicClient dynamic.Interface
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

	labels, err := metav1.LabelSelectorAsSelector(namespaceSelector)
	if err != nil {
		return false, err
	}
	options := metav1.ListOptions{
		LabelSelector: labels.String(),
	}

	// A list operation is required to use the namespace label selector.  Get all namespaces that
	// match the label selector and then check if the workload resource namespace matches one of the
	// returned namespaces.
	namespaces, err := w.KubeClient.CoreV1().Namespaces().List(context.TODO(), options)
	if err != nil {
		return false, err
	}

	for _, namespace := range namespaces.Items {
		// Namespace of workload resource must match
		if workload.GetNamespace() == namespace.Name {
			return true, nil
		}
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

	resource := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: pluralize.NewClient().Plural(strings.ToLower(workload.GetKind())),
	}

	labelSelector, err := metav1.LabelSelectorAsSelector(objectSelector)
	if err != nil {
		return false, err
	}
	options := metav1.ListOptions{
		LabelSelector: labelSelector.String(),
	}

	// Get the list of resources that match the object label selector
	objects, err := w.DynamicClient.Resource(resource).Namespace(workload.GetNamespace()).List(context.TODO(), options)
	if err != nil {
		return false, err
	}
	// Name of a returned object must match the workload resource name to have a match
	for _, object := range objects.Items {
		if object.GetName() == workload.GetName() {
			return true, nil
		}
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
