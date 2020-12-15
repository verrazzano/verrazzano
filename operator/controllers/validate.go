// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"errors"
	"fmt"
	installv1alpha1 "github.com/verrazzano/verrazzano/operator/api/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/operator/internal/util/env"
	"io/ioutil"
	"reflect"
	"sigs.k8s.io/yaml"
)

type chartVersion struct {
	APIVersion  string
	Description string
	Name        string
	Version     string
	AppVersion  string
}

// CompareVerrazzanoVersions Compare semantic version strings, returns < 0 if LHS is greater, 0 if identical, and > 1 if RHS greater
func CompareVerrazzanoVersions(currentVer string, newVer string) (result int, err error) {
	var currentVersion, newVersion *SemVersion
	if currentVersion, err = NewSemVersion(currentVer); err != nil {
		return 0, err
	}
	if newVersion, err = NewSemVersion(newVer); err != nil {
		return 0, err
	}
	return currentVersion.Compare(newVersion), nil
}

// ValidateVersion check that requestedVersion matches chart requestedVersion
func ValidateVersion(requestedVersion string) error {
	//chartVersion, err  := getCurrentChartVersion()
	//if err != nil {
	//	return err
	//}
	//val, err := CompareVerrazzanoVersions(requestedVersion, chartVersion.Version)
	//if err != nil {
	//	return err
	//}
	//if val != 0 {
	//	return fmt.Errorf("Requested version %s does not match chart version %s", requestedVersion, chartVersion.Version)
	//}
	return nil
}

// getCurrentChartVersion Load the current Chart.yaml into a chartVersion struct
func getCurrentChartVersion() (*chartVersion, error) {
	chartDir := env.VzChartDir()
	chartBytes, err := ioutil.ReadFile(chartDir + "/Chart.yaml")
	if err != nil {
		return nil, err
	}
	chartVersion := &chartVersion{}
	err = yaml.Unmarshal(chartBytes, chartVersion)
	if err != nil {
		return nil, err
	}
	return chartVersion, nil
}

// IsValidUpgradeRequest Ensure that for the upgrade case only the version field has changed
func IsValidUpgradeRequest(currentSpec *installv1alpha1.VerrazzanoSpec, newSpec *installv1alpha1.VerrazzanoSpec) error {
	// Validate the requested version
	err := ValidateVersion(currentSpec.Version)
	if err != nil {
		return err
	}

	// Verify that the new version request is > than the currently stored version
	versionCompareResult, err := CompareVerrazzanoVersions(currentSpec.Version, newSpec.Version)
	if err != nil {
		return err
	}
	if versionCompareResult <= 0 {
		return fmt.Errorf("Requested version #{newSpec.Version} is not greater than current version #{currentSpec.Version}")
	}
	// If any other field has changed from the stored spec return false
	if newSpec.Profile != currentSpec.Profile ||
		newSpec.EnvironmentName != currentSpec.EnvironmentName ||
		!reflect.DeepEqual(newSpec.Components, currentSpec.Components) {
		return errors.New("Configuration updates now allowed during upgrade between Verrazzano versions")
	}
	return nil
}
