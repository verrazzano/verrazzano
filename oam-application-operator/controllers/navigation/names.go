// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"fmt"
	"github.com/gertd/go-pluralize"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"strings"
)

// GetDefinitionOfResource converts APIVersion and Kind of CR to a CRD namespaced name.
// For example CR APIVersion.Kind core.oam.dev/v1alpha2.ContainerizedWorkload would be converted
// to containerizedworkloads.core.oam.dev in the default (i.e. "") namespace.
// resourceAPIVersion - The CR APIVersion
// resourceKind - The CR Kind
func GetDefinitionOfResource(resourceAPIVersion string, resourceKind string) types.NamespacedName {
	grp, ver := ParseGroupAndVersionFromAPIVersion(resourceAPIVersion)
	res := pluralize.NewClient().Plural(strings.ToLower(resourceKind))
	grpVerRes := metav1.GroupVersionResource{
		Group:    grp,
		Version:  ver,
		Resource: res,
	}
	var name string
	if grp == "" {
		name = grpVerRes.Resource
	} else {
		name = grpVerRes.Resource + "." + grpVerRes.Group
	}
	return types.NamespacedName{Namespace: "", Name: name}
}

// ParseGroupAndVersionFromAPIVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func ParseGroupAndVersionFromAPIVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// GetNamespacedNameFromObjectMeta creates a namespaced name from the values in an object meta.
func GetNamespacedNameFromObjectMeta(objectMeta metav1.ObjectMeta) types.NamespacedName {
	return types.NamespacedName{
		Namespace: objectMeta.Namespace,
		Name:      objectMeta.Name,
	}
}

// GetNamespacedNameFromUnstructured creates a namespaced name from the values in a unstructured.
func GetNamespacedNameFromUnstructured(u *unstructured.Unstructured) types.NamespacedName {
	return types.NamespacedName{
		Namespace: u.GetNamespace(),
		Name:      u.GetName(),
	}
}

// ParseNamespacedNameFromQualifiedName parses a string representation of a namespace qualified name.
func ParseNamespacedNameFromQualifiedName(qualifiedName string) (*types.NamespacedName, error) {
	parts := strings.SplitN(qualifiedName, "/", 2)
	if len(parts) < 2 {
		return nil, fmt.Errorf("failed to parse namespaced name %s", qualifiedName)
	}
	namespace := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if len(name) == 0 {
		return nil, fmt.Errorf("failed to parse namespaced name %s", qualifiedName)
	}
	return &types.NamespacedName{Namespace: namespace, Name: name}, nil
}
