// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package v1alpha1 contains API Schema definitions for the app v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=app.verrazzano.io
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// SchemeGroupVersion is the group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "app.verrazzano.io", Version: "v1alpha1"}

	// SchemeBuilder is used to add Go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
