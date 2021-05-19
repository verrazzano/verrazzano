// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package string

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
