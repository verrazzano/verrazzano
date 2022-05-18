// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"encoding/pem"
	"fmt"
	"io/fs"
	"os"
	"strings"

	vzos "github.com/verrazzano/verrazzano/pkg/os"

	"github.com/oracle/oci-go-sdk/v53/common"

	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"

	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type authenticationType string

const (
	// UserPrincipal is default auth type
	userPrincipal authenticationType = "user_principal"
	// InstancePrincipal is used for instance principle auth type
	instancePrincipal authenticationType = "instance_principal"
	// InstancePrincipalDelegationToken is used for instance principle delegation token auth type
	InstancePrincipalDelegationToken authenticationType = "instance_principle_delegation_token"
	// UnknownAuthenticationType is used for none meaningful auth type
	UnknownAuthenticationType authenticationType = "unknown_auth_type"
	// ValidateInProgress error message
	ValidateInProgressError = "Updates to resource not allowed while uninstall or upgrade is in progress"

	ociDNSSecretFileName            = "oci.yaml"
	fluentdOCISecretConfigEntry     = "config"
	fluentdOCISecretPrivateKeyEntry = "key"
	fluentdExpectedKeyPath          = "/root/.oci/key"
	fluentdOCIKeyFileEntry          = "key_file=/root/.oci/key"
	validateTempFilePattern         = "validate-"
)

// OCI DNS Secret Auth
type authData struct {
	Region      string             `json:"region"`
	Tenancy     string             `json:"tenancy"`
	User        string             `json:"user"`
	Key         string             `json:"key"`
	Fingerprint string             `json:"fingerprint"`
	AuthType    authenticationType `json:"authtype"`
}

// OCI DNS Secret Auth Wrapper
type ociAuth struct {
	Auth authData `json:"auth"`
}

func cleanTempFiles(log *zap.SugaredLogger) error {
	if err := vzos.RemoveTempFiles(log, validateTempFilePattern); err != nil {
		return fmt.Errorf("Error cleaning temp files: %s", err.Error())
	}
	return nil
}

// GetCurrentBomVersion Get the version string from the bom and return it as a semver object
func GetCurrentBomVersion() (*semver.SemVersion, error) {
	bom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return nil, err
	}
	v := bom.GetVersion()
	if v == constants.BomVerrazzanoVersion {
		// This branch is only hit when using a development BOM
		if len(os.Getenv("VZ_INSTALL_VERSION")) > 0 {
			v = os.Getenv("VZ_INSTALL_VERSION")
		} else {
			v = "1.0.1"
		}
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

// ValidateProfile check that requestedProfile is valid
func ValidateProfile(requestedProfile ProfileType) error {
	if len(requestedProfile) != 0 {
		switch requestedProfile {
		case Prod, Dev, ManagedCluster:
			return nil
		default:
			return fmt.Errorf("Requested profile %s is invalid, valid options are dev, prod, or managed-cluster",
				requestedProfile)
		}
	}
	return nil
}

// ValidateUpgradeRequest Ensures hat an upgrade is requested as part of an update if necessary,
// and that the version of an upgrade request is valid.
func ValidateUpgradeRequest(current *Verrazzano, new *Verrazzano) error {
	if !config.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}

	// Get the current BOM version
	bomVersion, err := GetCurrentBomVersion()
	if err != nil {
		return err
	}

	// Make sure the requested version matches what's in the BOM and is not < the current spec version
	newVerString := strings.TrimSpace(new.Spec.Version)
	if len(newVerString) > 0 {
		return validateNewVersion(current, newVerString, bomVersion)
	}

	// No new version set, we haven't done any upgrade before but may need to do one before allowing any edits;
	// this forces the user to opt-in to an upgrade before/with any other update
	if err := checkUpgradeRequired(strings.TrimSpace(current.Status.Version), bomVersion); err != nil {
		return err
	}
	return nil
}

//checkUpgradeRequired Returns an error if the current installed version is < the BOM version; if we're validating an
// update, this is an error condition, as we don't want to allow any updates without an upgrade
func checkUpgradeRequired(statusVersion string, bomVersion *semver.SemVersion) error {
	installedVerString := strings.TrimSpace(statusVersion)
	if len(installedVerString) == 0 {
		// Boundary condition -- likely just created and install hasn't started yet
		// - seems we get an immediate update/validation on initial CR creation
		return nil
	}
	installedVersion, err := semver.NewSemVersion(installedVerString)
	if err != nil {
		return err
	}
	if bomVersion.IsGreatherThan(installedVersion) {
		// Attempted an update before an upgrade has been done, reject the edit
		return fmt.Errorf("Upgrade required for update, set version field to v%v to upgrade", bomVersion.ToString())
	}
	return nil
}

func validateNewVersion(current *Verrazzano, newVerString string, bomVersion *semver.SemVersion) error {
	// Make sure the requested version matches what's in the BOM; we only have one version bundled at present
	newSpecVer, err := semver.NewSemVersion(newVerString)
	if err != nil {
		return err
	}
	if !newSpecVer.IsEqualTo(bomVersion) {
		// A newer version is available, the user must opt-in to an upgrade before we allow any edits
		return fmt.Errorf("Requested version %s does not match BOM version v%s, please upgrade to the current BOM version",
			newSpecVer.ToString(), bomVersion.ToString())
	}

	// Make sure this isn't a rollback attempt from the currently installed version, which is currently unsupported
	// - use case is, user rolls back to an earlier version of the platform operator and requests the older BOM version
	currentStatusVersion, err := semver.NewSemVersion(strings.TrimSpace(current.Status.Version))
	if err != nil {
		// for this path we should alwyas have a status version
		return err
	}
	if newSpecVer.IsLessThan(currentStatusVersion) {
		return fmt.Errorf("Requested version %s less than installed version %s, rollback is not supported",
			newSpecVer.ToString(), currentStatusVersion.ToString())
	}

	// Sanity check, verify that the new version request is > than the current spec version
	// - in reality, this should probably never happen unless we've introduced an error into the controller
	currentVerString := strings.TrimSpace(current.Spec.Version)
	if len(currentVerString) > 0 {
		currentSpecVer, err := semver.NewSemVersion(currentVerString)
		if err != nil {
			return err
		}
		if newSpecVer != nil && newSpecVer.IsLessThan(currentSpecVer) {
			return fmt.Errorf("Requested version %s is not newer than current version %s",
				newSpecVer.ToString(), currentSpecVer.ToString())
		}
	}
	// Simple update (spec edit at the same installed version)
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
	if old.Status.State == "" || old.Status.State == VzStateReady || old.Status.State == VzStateFailed || old.Status.State == VzStatePaused || old.Status.State == VzStateInstalling {
		return nil
	}
	return fmt.Errorf(ValidateInProgressError)
}

// validateOCISecrets - Validate that the OCI DNS and Fluentd OCI secrets required by install exists, if configured
func validateOCISecrets(client client.Client, spec *VerrazzanoSpec) error {
	if err := validateOCIDNSSecret(client, spec); err != nil {
		return err
	}
	if err := validateFluentdOCIAuthSecret(client, spec); err != nil {
		return err
	}
	return nil
}

func validateFluentdOCIAuthSecret(client client.Client, spec *VerrazzanoSpec) error {
	if spec.Components.Fluentd == nil || spec.Components.Fluentd.OCI == nil {
		return nil
	}
	apiSecretName := spec.Components.Fluentd.OCI.APISecret
	if len(apiSecretName) > 0 {
		secret := &corev1.Secret{}
		if err := getInstallSecret(client, apiSecretName, secret); err != nil {
			return err
		}
		// validate config secret
		if err := validateFluentdConfigData(secret); err != nil {
			return err
		}
		// Validate key data exists and is a valid pem format
		pemData, err := validateSecretKey(secret, fluentdOCISecretPrivateKeyEntry, nil)
		if err != nil {
			return err
		}
		if err := validatePrivateKey(secret.Name, pemData); err != nil {
			return err
		}
	}
	return nil
}

//validateFluentdConfigData - Validate the OCI config contents in the Fluentd secret
func validateFluentdConfigData(secret *corev1.Secret) error {
	secretName := secret.Name
	configData, ok := secret.Data[fluentdOCISecretConfigEntry]
	if !ok {
		return fmt.Errorf("Did not find OCI configuration in secret \"%s\"", secretName)
	}
	// Write the OCI config in the secret to a temp file and use the OCI SDK
	// ConfigurationProvider API to validate its contents
	configTemp, err := os.CreateTemp(os.TempDir(), validateTempFilePattern)
	if err != nil {
		return err
	}
	defer func() {
		os.Remove(configTemp.Name())
	}()
	const ociConfigErrorFormatString = "%s not specified in Fluentd OCI config secret \"%s\""
	if err := os.WriteFile(configTemp.Name(), configData, fs.ModeAppend); err != nil {
		return err
	}
	provider, err := common.ConfigurationProviderFromFile(configTemp.Name(), "")
	if err != nil {
		return err
	}
	userOCID, err := provider.UserOCID()
	if err != nil {
		return err
	}
	if len(userOCID) == 0 {
		return fmt.Errorf(ociConfigErrorFormatString, "User OCID", secretName)
	}
	tenancyOCID, err := provider.TenancyOCID()
	if err != nil {
		return err
	}
	if len(tenancyOCID) == 0 {
		return fmt.Errorf(ociConfigErrorFormatString, "Tenancy OCID", secretName)
	}
	fingerprint, err := provider.KeyFingerprint()
	if err != nil {
		return err
	}
	if len(fingerprint) == 0 {
		return fmt.Errorf(ociConfigErrorFormatString, "Fingerprint", secretName)
	}
	region, err := provider.Region()
	if err != nil {
		return err
	}
	if len(region) == 0 {
		return fmt.Errorf(ociConfigErrorFormatString, "Region", secretName)
	}
	if !strings.Contains(string(configData), fluentdOCIKeyFileEntry) {
		return fmt.Errorf("Unexpected or missing value for the Fluentd OCI key file location in secret \"%s\", should be \"%s\"",
			secretName, fluentdExpectedKeyPath)
	}
	return nil
}

func validateOCIDNSSecret(client client.Client, spec *VerrazzanoSpec) error {
	if spec.Components.DNS == nil || spec.Components.DNS.OCI == nil {
		return nil
	}
	secret := &corev1.Secret{}
	ociDNSConfigSecret := spec.Components.DNS.OCI.OCIConfigSecret
	if err := getInstallSecret(client, ociDNSConfigSecret, secret); err != nil {
		return err
	}
	// Verify that the oci secret has one value
	if len(secret.Data) != 1 {
		return fmt.Errorf("Secret \"%s\" for OCI DNS should have one data key, found %v", ociDNSConfigSecret, len(secret.Data))
	}
	for key := range secret.Data {
		// validate auth_type
		var authProp ociAuth
		if err := validateSecretContents(secret.Name, secret.Data[key], &authProp); err != nil {
			return err
		}
		if authProp.Auth.AuthType != instancePrincipal && authProp.Auth.AuthType != userPrincipal && authProp.Auth.AuthType != "" {
			return fmt.Errorf("Authtype \"%v\" in OCI secret must be either '%s' or '%s'", authProp.Auth.AuthType, userPrincipal, instancePrincipal)
		}
		if authProp.Auth.AuthType == userPrincipal {
			if err := validatePrivateKey(secret.Name, []byte(authProp.Auth.Key)); err != nil {
				return err
			}
		}
	}
	return nil
}

func getInstallSecret(client client.Client, secretName string, secret *corev1.Secret) error {
	err := client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: constants.VerrazzanoInstallNamespace}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("Secret \"%s\" must be created in the \"%s\" namespace before installing Verrrazzano", secretName, constants.VerrazzanoInstallNamespace)
		}
		return err
	}
	return nil
}

func validatePrivateKey(secretName string, pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("Private key in secret \"%s\" is either empty or not a valid key in PEM format", secretName)
	}
	return nil
}

func validateSecretKey(secret *corev1.Secret, dataKey string, target interface{}) ([]byte, error) {
	var secretBytes []byte
	var ok bool
	if secretBytes, ok = secret.Data[dataKey]; !ok {
		return nil, fmt.Errorf("Expected entry \"%s\" not found in secret \"%s\"", dataKey, secret.Name)
	}
	if target == nil {
		return secretBytes, nil
	}
	if err := validateSecretContents(secret.Name, secretBytes, target); err != nil {
		return secretBytes, nil
	}
	return secretBytes, nil
}

func validateSecretContents(secretName string, bytes []byte, target interface{}) error {
	if len(bytes) == 0 {
		return fmt.Errorf("Secret \"%s\" data is empty", secretName)
	}
	if err := yaml.Unmarshal(bytes, &target); err != nil {
		return err
	}
	return nil
}

//ValidateVersionHigherOrEqual checks that currentVersion matches requestedVersion or is a higher version
func ValidateVersionHigherOrEqual(currentVersion string, requestedVersion string) bool {
	log := zap.S().With("validate", "version")
	log.Info("Validate version")
	if len(requestedVersion) == 0 {
		log.Error("Invalid requestedVersion of length 0.")
		return false
	}

	if len(currentVersion) == 0 {
		log.Error("Invalid currentVersion of length 0.")
		return false
	}

	requestedSemVer, err := semver.NewSemVersion(requestedVersion)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid requestedVersion : %s, error: %v.", requestedVersion, err))
		return false
	}

	currentSemVer, err := semver.NewSemVersion(currentVersion)
	if err != nil {
		log.Error(fmt.Sprintf("Invalid currentVersion : %s, error: %v.", currentVersion, err))
		return false
	}

	return currentSemVer.IsEqualTo(requestedSemVer) || currentSemVer.IsGreatherThan(requestedSemVer)

}

// ValidateInstallOverrides checks that the overrides slice has only one override type per slice item
func ValidateInstallOverrides(Overrides []Overrides) error {
	overridePerItem := 0
	for _, override := range Overrides {
		if override.ConfigMapRef != nil {
			overridePerItem++
		}
		if override.SecretRef != nil {
			overridePerItem++
		}
		if overridePerItem > 1 {
			return fmt.Errorf("Invalid install overrides. Cannot specify more than one override type in the same list element")
		}
		if overridePerItem == 0 {
			return fmt.Errorf("Invalid install overrides. No override specified")
		}
		overridePerItem = 0
	}
	return nil
}
