// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmManagedResource provides an object type and name for a resource managed within a helm chart
type HelmManagedResource struct {
	Obj            clipkg.Object
	NamespacedName types.NamespacedName
}

const (
	helmReleaseNameKey      = "meta.helm.sh/release-name"
	helmReleaseNamespaceKey = "meta.helm.sh/release-namespace"
	helmResourcePolicyKey   = "helm.sh/resource-policy"
	managedByLabelKey       = "app.kubernetes.io/managed-by"
)

// AssociateHelmObject annotates an object as being managed by the specified release Helm chart
// If the object was already associated with a different Helm release (e.g verrazzano), then that relationship will be broken
func AssociateHelmObject(cli clipkg.Client, obj clipkg.Object, releaseName types.NamespacedName, namespacedName types.NamespacedName, keepResource bool) (clipkg.Object, error) {
	if err := cli.Get(context.TODO(), namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return obj, nil
		}
		return obj, err
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[helmReleaseNameKey] = releaseName.Name
	annotations[helmReleaseNamespaceKey] = releaseName.Namespace
	if keepResource {
		// Specify "keep" so that resource doesn't get deleted when we change the release name
		annotations[helmResourcePolicyKey] = "keep"
	}
	obj.SetAnnotations(annotations)
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[managedByLabelKey] = "Helm"
	obj.SetLabels(labels)
	err := cli.Update(context.TODO(), obj)
	return obj, err
}

// DisassociateHelmObject removes Helm release annotations from an object so that it is no longer
// managed by that Helm chart
func DisassociateHelmObject(cli clipkg.Client, obj clipkg.Object, namespacedName types.NamespacedName, keepResource bool, managedByHelm bool) (clipkg.Object, error) {
	if err := cli.Get(context.TODO(), namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return obj, nil
		}
		return obj, err
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	delete(annotations, helmReleaseNameKey)
	delete(annotations, helmReleaseNamespaceKey)

	if keepResource {
		// Specify "keep" so that resource doesn't get deleted when we change the release name
		annotations[helmResourcePolicyKey] = "keep"
	}
	obj.SetAnnotations(annotations)
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	if managedByHelm {
		labels[managedByLabelKey] = "Helm"
	} else {
		delete(labels, managedByLabelKey)
	}
	obj.SetLabels(labels)
	err := cli.Update(context.TODO(), obj)
	return obj, err
}

// RemoveResourcePolicyAnnotation removes the resource policy annotation to allow the resource to be managed by helm
func RemoveResourcePolicyAnnotation(cli clipkg.Client, obj clipkg.Object, namespacedName types.NamespacedName) (clipkg.Object, error) {
	if err := cli.Get(context.TODO(), namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return obj, nil
		}
		return obj, err
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return obj, nil
	}
	delete(annotations, helmResourcePolicyKey)
	obj.SetAnnotations(annotations)
	err := cli.Update(context.TODO(), obj)
	return obj, err
}
