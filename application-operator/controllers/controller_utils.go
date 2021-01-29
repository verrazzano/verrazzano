// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

// StringSliceContainsString determines if a string is found in a string slice.
// slice is the string slice to search. May be nil.
// find is the string to search for in the slice.
// Returns true if the string is found in the slice and false otherwise.
func StringSliceContainsString(slice []string, find string) bool {
	for _, s := range slice {
		if s == find {
			return true
		}
	}
	return false
}

// RemoveStringFromStringSlice removes a string from a string slice.
// slice is the string slice to remove the string from. May be nil.
// remove is the string to remove from the slice.
// Returns a new slice with the remove string removed.
func RemoveStringFromStringSlice(slice []string, remove string) []string {
	result := []string{}
	for _, s := range slice {
		if s == remove {
			continue
		}
		result = append(result, s)
	}
	return result
}
