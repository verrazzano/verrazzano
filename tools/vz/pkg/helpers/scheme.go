// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = vmcv1alpha1.AddToScheme(scheme)
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: clustersv1alpha1.SchemeGroupVersion.Group, Version: clustersv1alpha1.SchemeGroupVersion.Version, Kind: clustersv1alpha1.VerrazzanoProjectKind + "List"}, &clustersv1alpha1.VerrazzanoProjectList{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: vmcv1alpha1.SchemeGroupVersion.Group, Version: vmcv1alpha1.SchemeGroupVersion.Version, Kind: vmcv1alpha1.VerrazzanoManagedClusterKind + "List"}, &vmcv1alpha1.VerrazzanoManagedClusterList{})
	AddCapiToScheme(scheme)
	return scheme
}

func AddCapiToScheme(scheme *runtime.Scheme) {
	for _, resource := range capiNamespacedResources {
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: resource.GVR.Group, Version: resource.GVR.Version, Kind: resource.Kind + "List"}, &unstructured.Unstructured{})
	}
	for _, resource := range capiResources {
		scheme.AddKnownTypeWithName(schema.GroupVersionKind{Group: resource.GVR.Group, Version: resource.GVR.Version, Kind: resource.Kind + "List"}, &unstructured.Unstructured{})
	}
}
