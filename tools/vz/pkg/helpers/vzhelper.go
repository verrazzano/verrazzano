// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/semver"
	v1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/github"
	"io"
	adminv1 "k8s.io/api/admissionregistration/v1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
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

const defaultVerrazzano = `apiVersion: install.verrazzano.io/v1beta1
kind: Verrazzano
metadata:
  name: verrazzano
  namespace: default`

func NewDefaultVerrazzano() (client.Object, error) {
	vz := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(defaultVerrazzano), vz); err != nil {
		return nil, err
	}
	return vz, nil
}

func NewVerrazzanoForVersion(groupVersion schema.GroupVersion) func() interface{} {
	switch groupVersion {
	case v1alpha1.SchemeGroupVersion:
		return func() interface{} {
			return &v1alpha1.Verrazzano{}
		}
	default:
		return func() interface{} {
			return &v1beta1.Verrazzano{}
		}
	}
}

// FindVerrazzanoResource - find the single Verrazzano resource
func FindVerrazzanoResource(client client.Client) (*v1beta1.Verrazzano, error) {
	vzList := v1beta1.VerrazzanoList{}
	err := client.List(context.TODO(), &vzList)
	if err != nil {
		// If v1beta1 resource version doesn't exist, try v1alpha1
		if meta.IsNoMatchError(err) {
			return findVerazzanoResourceV1Alpha1(client)
		}
		return nil, failedToFindResourceError(err)
	}
	if err := checkListLength(len(vzList.Items)); err != nil {
		return nil, err
	}
	return &vzList.Items[0], nil
}

// GetVerrazzanoResource - get a Verrazzano resource
func GetVerrazzanoResource(client client.Client, namespacedName types.NamespacedName) (*v1beta1.Verrazzano, error) {
	vz := &v1beta1.Verrazzano{}
	if err := client.Get(context.TODO(), namespacedName, vz); err != nil {
		if meta.IsNoMatchError(err) {
			return getVerrazzanoResourceV1Alpha1(client, namespacedName)
		}
		return nil, failedToGetResourceError(err)

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
	_ = v1alpha1.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)
	_ = adminv1.SchemeBuilder.AddToScheme(scheme)
	_ = rbacv1.SchemeBuilder.AddToScheme(scheme)
	_ = appv1.SchemeBuilder.AddToScheme(scheme)
	return scheme
}

// GetNamespacesForAllComponents returns the list of unique namespaces of all the components included in the Verrazzano resource
func GetNamespacesForAllComponents(vz v1beta1.Verrazzano) []string {
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
func getAllComponents(vzRes v1beta1.Verrazzano) []string {
	var compSlice = make([]string, 0)

	for _, compStatusDetail := range vzRes.Status.Components {
		if compStatusDetail.State == v1beta1.CompStateDisabled {
			continue
		}
		compSlice = append(compSlice, compStatusDetail.Name)
	}
	return compSlice
}

func findVerazzanoResourceV1Alpha1(client client.Client) (*v1beta1.Verrazzano, error) {
	vzV1Alpha1List := v1alpha1.VerrazzanoList{}
	err := client.List(context.TODO(), &vzV1Alpha1List)
	if err != nil {
		return nil, failedToFindResourceError(err)
	}
	if err := checkListLength(len(vzV1Alpha1List.Items)); err != nil {
		return nil, err
	}
	vzConverted := &v1beta1.Verrazzano{}
	err = vzV1Alpha1List.Items[0].ConvertTo(vzConverted)
	return vzConverted, err
}

func getVerrazzanoResourceV1Alpha1(client client.Client, namespacedName types.NamespacedName) (*v1beta1.Verrazzano, error) {
	vzV1Alpha1 := &v1alpha1.Verrazzano{}
	if err := client.Get(context.TODO(), namespacedName, vzV1Alpha1); err != nil {
		return nil, failedToGetResourceError(err)
	}
	vz := &v1beta1.Verrazzano{}
	err := vzV1Alpha1.ConvertTo(vz)
	return vz, err
}

func failedToFindResourceError(err error) error {
	return fmt.Errorf("Failed to find any Verrazzano resources: %s", err.Error())
}

func failedToGetResourceError(err error) error {
	return fmt.Errorf("Failed to get a Verrazzano install resource: %s", err.Error())
}

func checkListLength(length int) error {
	if length == 0 {
		return fmt.Errorf("Failed to find any Verrazzano resources")
	}
	if length != 1 {
		return fmt.Errorf("Expected to only find one Verrazzano resource, but found %d", length)
	}
	return nil
}
