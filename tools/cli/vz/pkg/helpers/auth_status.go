// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"strings"
)

func LoggedIn() bool {
	return strings.Split(GetCurrentContextFromKubeConfig(), "@")[0] == "verrazzano"
}

func LoggedOut() bool {
	return strings.Split(GetCurrentContextFromKubeConfig(), "@")[0] != "verrazzano"
}

func RemoveAllAuthData() {
	// Remove the cluster with nickname verrazzano
	RemoveClusterFromKubeConfig("verrazzano")

	// Remove the user with nickname verrazzano
	RemoveUserFromKubeConfig("verrazzano")

	// Remove the currentcontext
	RemoveContextFromKubeConfig(GetCurrentContextFromKubeConfig())

	// Set currentcluster to the cluster before the user logged in
	SetCurrentContextInKubeConfig(strings.Split(GetCurrentContextFromKubeConfig(), "@")[1])

}
