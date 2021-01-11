// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import "reflect"

// QualifiedResourceRelation identifies a specific related resource
// (both APIVersion/Kind and namespace name) along this the role of the resource
// in the relationship.
type QualifiedResourceRelation struct {
	APIVersion string `json:"apiversion"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Role       string `json:"role"`
}

// QualifiedResourceRelationSlicesEquivalent determines if two slices of related resources are equivalent.
// The comparison does not depend on the order of the relations in the two slices.
// left - The first qualified resource relation for the equivalence comparison
// right - The second qualified resource relation for the equivalence comparison
func QualifiedResourceRelationSlicesEquivalent(left []QualifiedResourceRelation, right []QualifiedResourceRelation) bool {
	// Verify items in left slice exists in right slice
	for _, rel := range left {
		if !QualifiedResourceRelationsContain(right, &rel) {
			return false
		}
	}
	// Verify items in right slice exist in left slice
	for _, rel := range right {
		if !QualifiedResourceRelationsContain(left, &rel) {
			return false
		}
	}
	return true
}

// QualifiedResourceRelationsContain determines if a slice of relations contains a specific relation.
// slice - The slice of qualified resource relations to search
// find - The qualified resource relation to find in the slice
func QualifiedResourceRelationsContain(slice []QualifiedResourceRelation, find *QualifiedResourceRelation) bool {
	for _, rel := range slice {
		if reflect.DeepEqual(find, &rel) {
			return true
		}
	}
	return false
}
