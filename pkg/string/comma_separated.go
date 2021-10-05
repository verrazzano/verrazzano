// Copyright (C) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package string

import "strings"

// CommaSeparatedStringContains checks for a string in a comma separated list of strings
// commaSeparated is the comma separated string to search. May be nil.
// s is the string to search for.
// Returns true if the string is found and false otherwise.
func CommaSeparatedStringContains(commaSeparated string, s string) bool {
	split := strings.Split(commaSeparated, ",")
	found := false
	for _, ac := range split {
		if ac == s {
			found = true
			break
		}
	}
	return found
}

// AppendToCommaSeparatedString appends a string to a comma separated list.
// It first checks that the entry is not already part of the comma separated list.
// commaSeparated is the comma separated string to append into. May be empty.
// s is the string to add.
// Returns the updated comma separated string
func AppendToCommaSeparatedString(commaSeparated string, s string) string {
	// If the string is empty, return the value passed in
	if commaSeparated == "" {
		return s
	}

	// Check if the value is already contained in the comma separated list
	if CommaSeparatedStringContains(commaSeparated, s) {
		return commaSeparated
	}

	// Append the new value
	split := strings.Split(commaSeparated, ",")
	split = append(split, s)
	return strings.Join(split, ",")
}

// RemoveFromCommaSeparatedString removes a string from a comma separated list.
// It first checks that the entry is not already part of the comma separated list.
// commaSeparated is the comma separated list to remove from. May be empty.
// s is the string to remove.
// Returns the updated comma separated string
func RemoveFromCommaSeparatedString(commaSeparated string, s string) string {
	// If the string is empty, return the value passed in
	if commaSeparated == "" {
		return commaSeparated
	}

	// Check if the value is not contained in the comma separated list
	if !CommaSeparatedStringContains(commaSeparated, s) {
		return commaSeparated
	}

	// Remove the value
	split := strings.Split(commaSeparated, ",")
	split = RemoveStringFromSlice(split, s)
	return strings.Join(split, ",")
}
