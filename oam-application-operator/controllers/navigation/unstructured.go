// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	"fmt"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/oam-application-operator/apis/oam/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetKindOfUnstructured get the Kubernetes kind for an unstructured resource.
func GetKindOfUnstructured(u *unstructured.Unstructured) (string, error) {
	if u == nil {
		return "", fmt.Errorf("invalid unstructured reference")
	}
	kind, found := u.Object["kind"].(string)
	if !found {
		return "", fmt.Errorf("unstructured does not contain kind")
	}
	return kind, nil
}

// GetAPIVersionOfUnstructured gets the Kubernetes apiVersion of the unstructured resource.
func GetAPIVersionOfUnstructured(u *unstructured.Unstructured) (string, error) {
	if u == nil {
		return "", fmt.Errorf("invalid unstructured reference")
	}
	kind, found := u.Object["apiVersion"].(string)
	if !found {
		return "", fmt.Errorf("unstructured does not contain api version")
	}
	return kind, nil
}

// GetAPIVersionKindOfUnstructured gets the Kubernetes apiVersion.kind of the unstructured resource.
func GetAPIVersionKindOfUnstructured(u *unstructured.Unstructured) (string, error) {
	if u == nil {
		return "", fmt.Errorf("invalid unstructured reference")
	}
	version, err := GetAPIVersionOfUnstructured(u)
	if err != nil {
		return "", err
	}
	kind, err := GetKindOfUnstructured(u)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", version, kind), nil
}

// FetchUnstructuredChildResourcesByAPIVersionKinds find all of the child resource of specific kinds
// having a specific parent UID.  The child kinds are APIVersion and Kind
// (e.g. apps/v1.Deployment or v1.Service).  The objects of these resource kinds are listed
// and the ones having the correct parent UID are collected and accumulated and returned.
// This is used to collect a subset children of a particular parent object.
// ctx - The calling context
// namespace - The namespace to search for children objects
// parentUID - The parent UID a child must have to be included in the result.
// childResKinds - The set of resource kinds a child's resource kind must in to be included in the result.
func FetchUnstructuredChildResourcesByAPIVersionKinds(ctx context.Context, cli client.Reader, log logr.Logger, namespace string, parentUID types.UID, childResKinds []v1alpha2.ChildResourceKind) ([]*unstructured.Unstructured, error) {
	var childResources []*unstructured.Unstructured
	log.Info("Fetch children", "parent", parentUID)
	for _, childResKind := range childResKinds {
		resources := unstructured.UnstructuredList{}
		resources.SetAPIVersion(childResKind.APIVersion)
		resources.SetKind(childResKind.Kind)
		if err := cli.List(ctx, &resources, client.InNamespace(namespace), client.MatchingLabels(childResKind.Selector)); err != nil {
			log.Error(err, "Failed listing children")
			return nil, err
		}
		for i, item := range resources.Items {
			for _, owner := range item.GetOwnerReferences() {
				if owner.UID == parentUID {
					childResources = append(childResources, &resources.Items[i])
					break
				}
			}
		}
	}
	return childResources, nil
}

// FetchUnstructuredByReference fetches an unstructured using the namespace and name from a qualified resource relation.
func FetchUnstructuredByReference(ctx context.Context, cli client.Reader, log logr.Logger, reference vzapi.QualifiedResourceRelation) (*unstructured.Unstructured, error) {
	var uns unstructured.Unstructured
	uns.SetAPIVersion(reference.APIVersion)
	uns.SetKind(reference.Kind)
	key := client.ObjectKey{Name: reference.Name, Namespace: reference.Namespace}
	log.Info("Fetch related resource", "resource", key)
	if err := cli.Get(ctx, key, &uns); err != nil {
		return nil, err
	}
	return &uns, nil
}
