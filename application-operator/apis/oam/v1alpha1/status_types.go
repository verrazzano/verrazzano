// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import "reflect"

// QualifiedResourceRelation identifies a specific related resource.
type QualifiedResourceRelation struct {
	// API version of the related resource.
	APIVersion string `json:"apiversion"`
	// Kind of the related resource.
	Kind string `json:"kind"`
	// Name of the related resource.
	Name string `json:"name"`
	// Namespace of the related resource.
	Namespace string `json:"namespace"`
	// Role of the related resource.  For example, `Deployment`.
	Role string `json:"role"`
}

// QualifiedResourceRelationSlicesEquivalent determines if two slices of related resources are equivalent.
// The comparison does not depend on the order of the relations in the two slices.
// left - The first qualified resource relation for the equivalence comparison
// right - The second qualified resource relation for the equivalence comparison
func QualifiedResourceRelationSlicesEquivalent(left []QualifiedResourceRelation, right []QualifiedResourceRelation) bool {
	// Verify items in left slice exists in right slice
	for i := range left {
		if !QualifiedResourceRelationsContain(right, &left[i]) {
			return false
		}
	}
	// Verify items in right slice exist in left slice
	for i := range right {
		if !QualifiedResourceRelationsContain(left, &right[i]) {
			return false
		}
	}
	return true
}

// QualifiedResourceRelationsContain determines if a slice of relations contains a specific relation.
// slice - The slice of qualified resource relations to search
// find - The qualified resource relation to find in the slice
func QualifiedResourceRelationsContain(slice []QualifiedResourceRelation, find *QualifiedResourceRelation) bool {
	for i := range slice {
		if reflect.DeepEqual(find, &slice[i]) {
			return true
		}
	}
	return false
}
