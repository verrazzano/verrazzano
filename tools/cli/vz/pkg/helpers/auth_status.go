// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"strings"
)

// To be used as nick name for verrazzano related clusters,contexts,users,etc in kubeconfig
const NickNameInKubeConfig = "verrazzano"

// Assuming that the api call will take place within Buffer seconds from checking validity of token
const BufferTime = 10

// Helper function to find if the user is logged in
func LoggedIn() (bool, error) {
	var isLoggedIn bool
	currentContext, err := GetCurrentContextFromKubeConfig()
	if err != nil {
		return isLoggedIn, err
	}
	isLoggedIn = strings.Split(currentContext, "@")[0] == NickNameInKubeConfig
	return isLoggedIn, nil
}

// Helper function to find if the user is logged out
func LoggedOut() (bool, error) {
	var isLoggedOut bool
	currentContext, err := GetCurrentContextFromKubeConfig()
	if err != nil {
		return isLoggedOut, err
	}
	isLoggedOut = strings.Split(currentContext, "@")[0] != NickNameInKubeConfig
	return isLoggedOut, nil
}

// Helper function that removes all the user details from kubeconfig
func RemoveAllAuthData() error {
	// Remove the cluster with nickname verrazzano
	err := RemoveClusterFromKubeConfig("verrazzano")
	if err != nil {
		return err
	}

	// Remove the user with nickname verrazzano
	err = RemoveUserFromKubeConfig("verrazzano")
	if err != nil {
		return err
	}

	currentContext, err := GetCurrentContextFromKubeConfig()
	if err != nil {
		return err
	}
	// Remove the currentcontext
	err = RemoveContextFromKubeConfig(currentContext)
	if err != nil {
		return err
	}

	// Set currentcluster to the cluster before the user logged in
	err = SetCurrentContextInKubeConfig(strings.Split(currentContext, "@")[1])
	if err != nil {
		return err
	}
	return nil
}
