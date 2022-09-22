// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validators

import (
	"context"
	"encoding/pem"
	goerrors "errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/onsi/gomega/gstruct/errors"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	validateTempFilePattern     = "validate-"
	FluentdOCISecretConfigEntry = "config"
	fluentdOCIKeyFileEntry      = "key_file=/root/.oci/key"
	fluentdExpectedKeyPath      = "/root/.oci/key"
	// UserPrincipal is default auth type
	UserPrincipal AuthenticationType = "user_principal"
	// InstancePrincipal is used for instance principle auth type
	InstancePrincipal       AuthenticationType = "instance_principal"
	ValidateInProgressError                    = "Updates to resource not allowed while uninstall or upgrade is in progress"

	FluentdOCISecretPrivateKeyEntry = "key"
	OciDNSSecretFileName            = "oci.yaml"
	// InstancePrincipalDelegationToken is used for instance principle delegation token auth type
	InstancePrincipalDelegationToken AuthenticationType = "instance_principle_delegation_token"
	// UnknownAuthenticationType is used for none meaningful auth type
	UnknownAuthenticationType AuthenticationType = "unknown_auth_type"
)

type AuthenticationType string

// OCI DNS Secret Auth
type AuthData struct {
	Region      string             `json:"region"`
	Tenancy     string             `json:"tenancy"`
	User        string             `json:"user"`
	Key         string             `json:"key"`
	Fingerprint string             `json:"fingerprint"`
	AuthType    AuthenticationType `json:"authtype"`
}

// OCI DNS Secret Auth Wrapper
type OciAuth struct {
	Auth AuthData `json:"auth"`
}

func CleanTempFiles(log *zap.SugaredLogger) error {
	if err := vzos.RemoveTempFiles(log, validateTempFilePattern); err != nil {
		return fmt.Errorf("Error cleaning temp files: %s", err.Error())
	}
	return nil
}

// combineErrors combines multiple errors into one error, nil if no error
func CombineErrors(errs []error) error {
	if len(errs) > 0 {
		return goerrors.New(errors.AggregateError(errs).Error())
	}
	return nil
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

func ValidateNewVersion(currStatusVerString string, currSpecVerString string, newVerString string, bomVersion *semver.SemVersion) error {
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
	currentStatusVersion, err := semver.NewSemVersion(strings.TrimSpace(currStatusVerString))
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
	currentVerString := strings.TrimSpace(currSpecVerString)
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

//checkUpgradeRequired Returns an error if the current installed version is < the BOM version; if we're validating an
// update, this is an error condition, as we don't want to allow any updates without an upgrade
func CheckUpgradeRequired(statusVersion string, bomVersion *semver.SemVersion) error {
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

func GetInstallSecret(client client.Client, secretName string, secret *corev1.Secret) error {
	err := client.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: constants.VerrazzanoInstallNamespace}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("Secret \"%s\" must be created in the \"%s\" namespace before installing Verrrazzano", secretName, constants.VerrazzanoInstallNamespace)
		}
		return err
	}
	return nil
}

func ValidateSecretContents(secretName string, bytes []byte, target interface{}) error {
	if len(bytes) == 0 {
		return fmt.Errorf("Secret \"%s\" data is empty", secretName)
	}
	if err := yaml.Unmarshal(bytes, &target); err != nil {
		return err
	}
	return nil
}

func ValidatePrivateKey(secretName string, pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("Private key in secret \"%s\" is either empty or not a valid key in PEM format", secretName)
	}
	return nil
}

//validateFluentdConfigData - Validate the OCI config contents in the Fluentd secret
func ValidateFluentdConfigData(secret *corev1.Secret) error {
	secretName := secret.Name
	configData, ok := secret.Data[FluentdOCISecretConfigEntry]
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

func ValidateSecretKey(secret *corev1.Secret, dataKey string, target interface{}) ([]byte, error) {
	var secretBytes []byte
	var ok bool
	if secretBytes, ok = secret.Data[dataKey]; !ok {
		return nil, fmt.Errorf("Expected entry \"%s\" not found in secret \"%s\"", dataKey, secret.Name)
	}
	if target == nil {
		return secretBytes, nil
	}
	if err := ValidateSecretContents(secret.Name, secretBytes, target); err != nil {
		return secretBytes, nil
	}
	return secretBytes, nil
}

// ValidateUpgradeRequest Ensures hat an upgrade is requested as part of an update if necessary,
// and that the version of an upgrade request is valid.
func ValidateUpgradeRequest(newSpecVerString string, currStatusVerString string, currSpecVerString string) error {
	if !config.Get().VersionCheckEnabled {
		zap.S().Infof("Version validation disabled")
		return nil
	}

	// Get the current BOM version
	bomVersion, err := GetCurrentBomVersion()
	if err != nil {
		return err
	}

	//// Make sure the requested version matches what's in the BOM and is not < the current spec version
	if len(newSpecVerString) > 0 {
		return ValidateNewVersion(currStatusVerString, currSpecVerString, newSpecVerString, bomVersion)
	}

	// No new version set, we haven't done any upgrade before but may need to do one before allowing any edits;
	// this forces the user to opt-in to an upgrade before/with any other update
	if err := CheckUpgradeRequired(strings.TrimSpace(currStatusVerString), bomVersion); err != nil {
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

// getClient returns a controller runtime client for the Verrazzano resource
func GetClient(scheme *runtime.Scheme) (client.Client, error) {

	config, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{Scheme: scheme})
}

// IsKubernetesVersionSupported verifies if Kubernetes version of cluster is supported
func IsKubernetesVersionSupported() bool {
	log := zap.S().With("validate", "kubernetes", "version")
	bom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		log.Errorf("error while reading the bom %v", err.Error())
		return false
	}

	supportedKubernetesVersions := bom.GetSupportedKubernetesVersion()
	if len(supportedKubernetesVersions) == 0 {
		log.Info("supported kubernetes versions not specified in the bom, assuming supports all versions")
		return true
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		log.Errorf("error while getting kubernetes client config %v", err.Error())
		return false
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("error while getting kubernetes client %v", err.Error())
		return false
	}

	versionInfo, err := client.ServerVersion()
	if err != nil {
		log.Errorf("error while getting kubernetes version %v", err.Error())
		return false
	}

	kubernetesVersion, err := semver.NewSemVersion(versionInfo.String())
	if err != nil {
		log.Errorf("invalid kubernetes version %s, error %v", versionInfo.String(), err.Error())
		return false
	}

	for _, supportedVersion := range supportedKubernetesVersions {
		version, err := semver.NewSemVersion(supportedVersion)
		if err != nil {
			log.Errorf("invalid supported kubernetes version %s, error %v", supportedVersion, err.Error())
			continue
		}

		if kubernetesVersion.IsEqualToOrPatchVersionOf(version) {
			return true
		}
	}

	log.Errorf("kubernetes version %s not supported, supported versions are %v", kubernetesVersion.ToString(), supportedKubernetesVersions)
	return false
}
