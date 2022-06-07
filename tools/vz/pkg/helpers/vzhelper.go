// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/github"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VZHelper interface {
	GetOutputStream() io.Writer
	GetErrorStream() io.Writer
	GetInputStream() io.Reader
	GetClient(cmd *cobra.Command) (client.Client, error)
	GetKubeClient(cmd *cobra.Command) (*kubernetes.Clientset, error)
}

// FindVerrazzanoResource - find the single Verrazzano resource
func FindVerrazzanoResource(client client.Client) (*vzapi.Verrazzano, error) {

	vzList := vzapi.VerrazzanoList{}
	err := client.List(context.TODO(), &vzList)
	if err != nil {
		return nil, err
	}
	if len(vzList.Items) == 0 {
		return nil, fmt.Errorf("Failed to find any Verrazzano resources")
	}
	if len(vzList.Items) != 1 {
		return nil, fmt.Errorf("Expected to only find one Verrazzano resource, but found %d", len(vzList.Items))
	}
	return &vzList.Items[0], nil
}

// GetVerrazzanoResource - get a Verrazzano resource
func GetVerrazzanoResource(client client.Client, namespacedName types.NamespacedName) (*vzapi.Verrazzano, error) {

	vz := &vzapi.Verrazzano{}
	err := client.Get(context.TODO(), namespacedName, vz)
	if err != nil {
		return nil, err
	}
	return vz, nil
}

// GetLatestReleaseVersion - get the version of the latest release of Verrazzano
func GetLatestReleaseVersion() (string, error) {
	// Get the list of all Verrazzano releases
	releases, err := github.ListReleases()
	if err != nil {
		return "", fmt.Errorf("Failed to get list of Verrazzano releases: %s", err.Error())
	}
	return getLatestReleaseVersion(releases)
}

// getLatestReleaseVersion - determine which release it the latest based on semver values
func getLatestReleaseVersion(releases []string) (string, error) {
	var latestRelease *semver.SemVersion
	for _, tag := range releases {
		tagSemver, err := semver.NewSemVersion(tag)
		if err != nil {
			return "", err
		}
		if latestRelease == nil {
			// Initialize with the first tag
			latestRelease = tagSemver
		} else {
			if tagSemver.IsGreatherThan(latestRelease) {
				// Update the latest release found
				latestRelease = tagSemver
			}
		}
	}
	return fmt.Sprintf("v%s", latestRelease.ToString()), nil
}

// GetWaitTimeout returns the time to wait for a command to complete
func GetWaitTimeout(cmd *cobra.Command) (time.Duration, error) {
	// Get the wait value from the command line
	wait, err := cmd.PersistentFlags().GetBool(constants.WaitFlag)
	if err != nil {
		return time.Duration(0), err
	}
	if wait {
		timeout, err := cmd.PersistentFlags().GetDuration(constants.TimeoutFlag)
		if err != nil {
			return time.Duration(0), err
		}
		return timeout, nil
	}

	// Return duration of zero since --wait=false was specified
	return time.Duration(0), nil
}
