package common

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
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	validateTempFilePattern     = "validate-"
	fluentdOCISecretConfigEntry = "config"
	fluentdOCIKeyFileEntry      = "key_file=/root/.oci/key"
	fluentdExpectedKeyPath      = "/root/.oci/key"
)

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

// verifyPlatformOperatorSingleton Verifies that only one instance of the VPO is running; when upgrading operators,
// if the terminationGracePeriod for the pod is > 0 there's a chance that an old version may try to handle resource
// updates before terminating.  In the longer term we may want some kind of leader-election strategy to support
// multiple instances, if that makes sense.
func VerifyPlatformOperatorSingleton(runtimeClient client.Client) error {
	var podList v1.PodList
	runtimeClient.List(context.TODO(), &podList,
		client.InNamespace(constants.VerrazzanoInstallNamespace),
		client.MatchingLabels{"app": "verrazzano-platform-operator"})
	if len(podList.Items) > 1 {
		return fmt.Errorf("Found more than one running instance of the platform operator, only one instance allowed")
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

func ValidateSecretKey(secret *corev1.Secret, dataKey string, target interface{}) ([]byte, error) {
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
