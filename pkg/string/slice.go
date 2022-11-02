// Copyright (C) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package string

import (
	"github.com/google/go-cmp/cmp"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"sort"
)

// SliceContainsString checks for a string in a slice of strings
// slice is the string slice to search. May be nil.
// s is the string to search for in the slice.
// Returns true if the string is found in the slice and false otherwise.
func SliceContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveStringFromSlice removes a string from a string slice.
// slice is the string slice to remove the string from. May be nil.
// s is the string to remove from the slice.
// Returns a new slice with the remove string removed.
func RemoveStringFromSlice(slice []string, s string) []string {
	result := []string{}
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return result
}

// UnorderedEqual checks if a map and array have the same string elements.
// The same order is not required.
func UnorderedEqual(mapBool map[string]bool, arrayStr []string) bool {
	if len(mapBool) != len(arrayStr) {
		return false
	}
	for element := range mapBool {
		if !SliceContainsString(arrayStr, element) {
			return false
		}
	}
	return true
}

// SliceToSet converts a slice of strings to a set of strings
func SliceToSet(list []string) map[string]bool {
	outSet := make(map[string]bool)
	for _, f := range list {
		outSet[f] = true
	}
	return outSet
}

// SliceAddString Adds a string to a slice if it is not already present
func SliceAddString(slice []string, s string) ([]string, bool) {
	if !SliceContainsString(slice, s) {
		return append(slice, s), true
	}
	return slice, false
}

// compareSlices compares 2 string slices after sorting
func CompareTwoSlices(log vzlog.VerrazzanoLogger, slice1 []string, slice2 []string) bool {
	sort.Strings(slice1)
	sort.Strings(slice2)

	if len(slice1) != len(slice2) {
		log.Errorf("%s doesn't match with %s", slice1, slice2)
		log.Infof(cmp.Diff(slice1, slice2))
		return false
	}
	for i, v := range slice1 {
		if v != slice2[i] {
			log.Errorf("%s doesn't match with %s", slice1, slice2)
			log.Infof(cmp.Diff(slice1, slice2))
			return false
		}
	}
	return true
}
