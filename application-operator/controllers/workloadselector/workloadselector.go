// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package workloadselector

import (
	"context"
	"strings"

	"github.com/gertd/go-pluralize"
	corev1 "k8s.io/api/core/v1"
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
func (w *WorkloadSelector) DoesWorkloadMatch(workload *unstructured.Unstructured, namespaceSelector *metav1.LabelSelector, objectSelector *metav1.LabelSelector, apiGroups []string, apiVersions []string, apiKinds []string) (bool, error) {
	// Get namespaces that match the given namespace label selector
	namespaces, err := w.getMatchingNamespaces(namespaceSelector)
	if err != nil {
		return false, err
	}

	// If no namespaces match then no need for any other processing so return no match
	if len(namespaces.Items) == 0 {
		return false, nil
	}

	return w.doesObjectMatch(workload, namespaces, objectSelector, apiGroups, apiVersions, apiKinds)
}

// getMatchingNamespaces returns a list of namespaces matching the specified namespace label selector
func (w *WorkloadSelector) getMatchingNamespaces(namespaceSelector *metav1.LabelSelector) (*corev1.NamespaceList, error) {
	labels, err := metav1.LabelSelectorAsSelector(namespaceSelector)
	if err != nil {
		return nil, err
	}
	options := metav1.ListOptions{
		LabelSelector: labels.String(),
	}

	namespaceList, err := w.KubeClient.CoreV1().Namespaces().List(context.TODO(), options)
	return namespaceList, err
}

// doesObjectMatch returns a boolean indicating whether an unstructured resource matches the criteria for a match.
// The criteria used to match is a list of namespaces, object label selector, and group, version, and kind values
func (w *WorkloadSelector) doesObjectMatch(workload *unstructured.Unstructured, namespaces *corev1.NamespaceList, objectSelector *metav1.LabelSelector, apiGroups []string, apiVersions []string, apiKinds []string) (bool, error) {
	// Get the group and version of the workload resource
	gv, err := schema.ParseGroupVersion(workload.GetAPIVersion())
	if err != nil {
		return false, nil
	}

	// Check that the workload resource GVK matches expected GVKs
	if !checkMatch(gv.Version, apiVersions) || !checkMatch(gv.Group, apiGroups) || !checkMatch(workload.GetKind(), apiKinds) {
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

	for _, namespace := range namespaces.Items {
		// Namespace of workload resource must match
		if workload.GetNamespace() != namespace.Name {
			continue
		}
		// Get the list of resources that match the object label selector
		objects, err := w.DynamicClient.Resource(resource).Namespace(namespace.Name).List(context.TODO(), options)
		if err != nil {
			return false, err
		}
		// Name of a returned object must match the workload resource name to have a match
		for _, object := range objects.Items {
			if object.GetName() == workload.GetName() {
				return true, nil
			}
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
