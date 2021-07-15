// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"strings"
)

func LoggedIn() bool {
	if strings.Split(GetCurrentContextFromKubeConfig(), "@")[0] == "verrazzano" {
		return true
	}
	return false
}

func LoggedOut() bool {
	if strings.Split(GetCurrentContextFromKubeConfig(), "@")[0] != "verrazzano" {
		return true
	}
	return false
}

func RemoveAllAuthData() {
	// Remove the cluster with nickname verrazzano
	RemoveClusterFromKubeConfig("verrazzano")

	// Remove the user with nickname verrazzano
	RemoveUserFromKubeConfig("verrazzano")

	// Remove the currentcontext
	RemoveUserFromKubeConfig(GetCurrentContextFromKubeConfig())

	// Set currentcluster to the cluster before the user logged in
	SetCurrentContextInKubeConfig(strings.Split(GetCurrentContextFromKubeConfig(), "@")[1])
}