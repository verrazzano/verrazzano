// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"

	vzconf "github.com/verrazzano/verrazzano/operator/config"
	vzcomp "github.com/verrazzano/verrazzano/operator/internal/component"
	"github.com/verrazzano/verrazzano/operator/internal/util/semver"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type chartVersion struct {
	APIVersion  string
	Description string
	Name        string
	Version     string
	AppVersion  string
}

// For unit test purposes
var readFileFunction = ioutil.ReadFile

// getCurrentChartVersion Load the current Chart.yaml into a chartVersion struct
func getCurrentChartVersion() (*semver.SemVersion, error) {
	chartDir := vzcomp.VzChartDir()
	chartBytes, err := readFileFunction(chartDir + "/Chart.yaml")
	if err != nil {
		return nil, err
	}
	chartVersion := &chartVersion{}
	err = yaml.Unmarshal(chartBytes, chartVersion)
	if err != nil {
		return nil, err
	}
	return semver.NewSemVersion(fmt.Sprintf("v%s", chartVersion.Version))
}

// ValidateVersion check that requestedVersion matches chart requestedVersion
func ValidateVersion(requestedVersion string) error {
	if !vzconf.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}
	if len(requestedVersion) == 0 {
		return nil
	}
	requestedSemVer, err := semver.NewSemVersion(requestedVersion)
	if err != nil {
		return err
	}
	chartSemVer, err := getCurrentChartVersion()
	if err != nil {
		return err
	}
	if !requestedSemVer.IsEqualTo(chartSemVer) {
		return fmt.Errorf("Requested version %s does not match chart version %s", requestedVersion, chartSemVer.ToString())
	}
	return nil
}

// ValidateUpgradeRequest Ensures that for the upgrade case only the version field has changed
func ValidateUpgradeRequest(currentSpec *VerrazzanoSpec, newSpec *VerrazzanoSpec) error {
	if !vzconf.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}
	// Short-circuit if the version strings are the same
	if currentSpec.Version == newSpec.Version {
		return nil
	}
	if len(newSpec.Version) == 0 {
		// if we get here, the current version is not empty, but the new version is
		return fmt.Errorf("Requested version is not specified")
	}
	if err := ValidateVersion(newSpec.Version); err != nil {
		// new version is not nil, but we couldn't parse it
		return err
	}

	requestedSemVer, err := semver.NewSemVersion(newSpec.Version)
	if err != nil {
		// parse error on new version string
		return err
	}

	// Verify that the new version request is > than the currently version
	if len(currentSpec.Version) > 0 {
		currentSemVer, err := semver.NewSemVersion(currentSpec.Version)
		if err != nil {
			// Unable to parse the current spec version; this should never happen
			return err
		}
		if requestedSemVer.IsLessThan(currentSemVer) {
			return fmt.Errorf("Requested version %s is not newer than current version %s", requestedSemVer.ToString(), currentSemVer.ToString())
		}
	}

	// If any other field has changed from the stored spec return false
	if newSpec.Profile != currentSpec.Profile ||
		newSpec.EnvironmentName != currentSpec.EnvironmentName ||
		!reflect.DeepEqual(newSpec.Components, currentSpec.Components) {
		return errors.New("Configuration updates not allowed during upgrade between Verrazzano versions")
	}
	return nil
}

// ValidateActiveInstall enforces that only one install of Verrazzano is allowed.
func ValidateActiveInstall(client client.Client) error {
	vzList := &VerrazzanoList{}

	err := client.List(context.Background(), vzList)
	if err != nil {
		return err
	}

	if len(vzList.Items) != 0 {
		return fmt.Errorf("Only one install of Verrazzano is allowed")
	}

	return nil
}

// ValidateInProgress makes sure there is not an install, uninstall or upgrade in progress
func ValidateInProgress(state StateType) error {
	if state == Installing || state == Uninstalling || state == Upgrading {
		return fmt.Errorf("Updates to resource not allowed while install, uninstall or upgrade is in progress")
	}

	return nil
}
