// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"strings"

	"github.com/verrazzano/verrazzano/application-operator/constants"
)

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

// ConvertAPIVersionToGroupAndVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func ConvertAPIVersionToGroupAndVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// IsWorkloadMarkedForUpgrade checks to see if a workload needs to be upgraded to the latest
// Verrazzano version. Verrazzano defines some resources which are used by applications and when Verrazzano is upgraded,
// a user can mark an application to indicate that it should use the latest resources defined by Verrazzano.
// A response of 'true' indicates that reconcile should use the latest resources defined by Verrazzano and a response
// of 'false' indicates that the application should continue to use the current values.
func IsWorkloadMarkedForUpgrade(labels map[string]string, previousUpgrade string) bool {
	return labels[constants.LabelUpgradeVersion] != previousUpgrade
}
