// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/github"
	"io"
	adminv1 "k8s.io/api/admissionregistration/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VZHelper interface {
	GetOutputStream() io.Writer
	GetErrorStream() io.Writer
	GetInputStream() io.Reader
	GetClient(cmd *cobra.Command) (client.Client, error)
	GetKubeClient(cmd *cobra.Command) (kubernetes.Interface, error)
	GetHTTPClient() *http.Client
	GetDynamicClient(cmd *cobra.Command) (dynamic.Interface, error)
}

// FindVerrazzanoResource - find the single Verrazzano resource
func FindVerrazzanoResource(client client.Client) (*vzapi.Verrazzano, error) {

	vzList := vzapi.VerrazzanoList{}
	err := client.List(context.TODO(), &vzList)
	if err != nil {
		return nil, fmt.Errorf("Failed to find any Verrazzano resources: %s", err.Error())
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
		return nil, fmt.Errorf("Failed to get a Verrazzano install resource: %s", err.Error())
	}
	return vz, nil
}

// GetLatestReleaseVersion - get the version of the latest release of Verrazzano
func GetLatestReleaseVersion(client *http.Client) (string, error) {
	// Get the list of all Verrazzano releases
	releases, err := github.ListReleases(client)
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

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)
	_ = adminv1.SchemeBuilder.AddToScheme(scheme)
	_ = rbacv1.SchemeBuilder.AddToScheme(scheme)
	_ = appv1.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

// GetNamespacesForAllComponents returns the list of unique namespaces of all the components included in the Verrazzano resource
func GetNamespacesForAllComponents(vz vzapi.Verrazzano) []string {
	allComponents := getAllComponents(vz)
	var nsList []string
	for _, eachComp := range allComponents {
		nsList = append(nsList, constants.ComponentNameToNamespacesMap[eachComp]...)
	}
	if len(nsList) > 0 {
		nsList = RemoveDuplicate(nsList)
	}
	return nsList
}

// getAllComponents returns the list of components from the Verrazzano resource
func getAllComponents(vzRes vzapi.Verrazzano) []string {
	var compSlice = make([]string, 0)

	for _, compStatusDetail := range vzRes.Status.Components {
		if compStatusDetail.State == vzapi.CompStateNotInstalled {
			continue
		}
		compSlice = append(compSlice, compStatusDetail.Name)
	}
	return compSlice
}
