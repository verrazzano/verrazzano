// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"reflect"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/semver"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetCurrentBomVersion Get the version string from the bom and return it as a semver object
func GetCurrentBomVersion() (*semver.SemVersion, error) {
	bom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	v := bom.GetVersion()
	if v == constants.BomVerrazzanoVersion {
		// This will only happen during development testing, the value doesn't matter
		v = "1.0.1"
	}
	return semver.NewSemVersion(fmt.Sprintf("v%s", v))
}

// ValidateVersion check that requestedVersion matches BOM requestedVersion
func ValidateVersion(requestedVersion string) error {
	if !config.Get().VersionCheckEnabled {
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
	bomSemVer, err := GetCurrentBomVersion()
	if err != nil {
		return err
	}
	if !requestedSemVer.IsEqualTo(bomSemVer) {
		return fmt.Errorf("Requested version %s does not match BOM version %s", requestedSemVer.ToString(), bomSemVer.ToString())
	}
	return nil
}

// ValidateUpgradeRequest Ensures that for the upgrade case only the version field has changed
func ValidateUpgradeRequest(currentSpec *VerrazzanoSpec, newSpec *VerrazzanoSpec) error {
	if !config.Get().VersionCheckEnabled {
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
func ValidateInProgress(old *Verrazzano, new *Verrazzano) error {
	if old.Status.State == Ready {
		return nil
	}

	// Allow enable component during install
	if old.Status.State == Installing {
		if coherence.IsEnabled(new.Spec.Components.Coherence) && !coherence.IsEnabled(old.Spec.Components.Coherence) {
			return nil
		}
		if weblogic.IsEnabled(new.Spec.Components.WebLogic) && !weblogic.IsEnabled(old.Spec.Components.WebLogic) {
			return nil
		}
	}
	return fmt.Errorf("Updates to resource not allowed while install, uninstall or upgrade is in progress")
}

// ValidateOciDNSSecret makes sure that the OCI DNS secret required by install exists
func ValidateOciDNSSecret(client client.Client, spec *VerrazzanoSpec) error {
	if spec.Components.DNS != nil && spec.Components.DNS.OCI != nil {
		secret := &corev1.Secret{}
		err := client.Get(context.TODO(), types.NamespacedName{Name: spec.Components.DNS.OCI.OCIConfigSecret, Namespace: constants.VerrazzanoInstallNamespace}, secret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return fmt.Errorf("The secret \"%s\" must be created in the %s namespace before installing Verrrazzano for OCI DNS", spec.Components.DNS.OCI.OCIConfigSecret, constants.VerrazzanoInstallNamespace)
			}
			return err
		}
	}

	return nil
}
