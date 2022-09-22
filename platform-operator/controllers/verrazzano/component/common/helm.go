// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// HelmManagedResource provides an object type and name for a resource managed within a helm chart
type HelmManagedResource struct {
	Obj            controllerutil.Object
	NamespacedName types.NamespacedName
}

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
	annotations["meta.helm.sh/release-name"] = releaseName.Name
	annotations["meta.helm.sh/release-namespace"] = releaseName.Namespace
	if keepResource {
		// Specify "keep" so that resource doesn't get deleted when we change the release name
		annotations["helm.sh/resource-policy"] = "keep"
	}
	obj.SetAnnotations(annotations)
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app.kubernetes.io/managed-by"] = "Helm"
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
	delete(annotations, "helm.sh/resource-policy")
	obj.SetAnnotations(annotations)
	err := cli.Update(context.TODO(), obj)
	return obj, err
}

// RemoveAllHelmAnnotationsAndLabels removes all helm annotations and labels
func RemoveAllHelmAnnotationsAndLabels(cli clipkg.Client, obj clipkg.Object, namespacedName types.NamespacedName) (clipkg.Object, error) {
	if err := cli.Get(context.TODO(), namespacedName, obj); err != nil {
		if errors.IsNotFound(err) {
			return obj, nil
		}
		return obj, err
	}
	// Clear Helm annotations
	annotations := obj.GetAnnotations()
	if annotations != nil {
		delete(annotations, "helm.sh/resource-policy")
		delete(annotations, "meta.helm.sh/release-name")
		delete(annotations, "meta.helm.sh/release-namespace")
		obj.SetAnnotations(annotations)
	}

	// Clear Helm labels
	labels := obj.GetLabels()
	if labels != nil {
		delete(annotations, "app.kubernetes.io/managed-by")
		obj.SetLabels(labels)
	}

	err := cli.Update(context.TODO(), obj)
	return obj, err
}
