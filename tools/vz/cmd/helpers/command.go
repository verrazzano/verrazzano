// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"fmt"
	"helm.sh/helm/v3/pkg/strvals"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// NewCommand - utility method to create cobra commands
func NewCommand(vzHelper helpers.VZHelper, use string, short string, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
	}

	// Configure the IO streams
	cmd.SetOut(vzHelper.GetOutputStream())
	cmd.SetErr(vzHelper.GetErrorStream())
	cmd.SetIn(vzHelper.GetInputStream())

	// Disable usage output on errors
	cmd.SilenceUsage = true
	return cmd
}

// GetWaitTimeout returns the time to wait for a command to complete
func GetWaitTimeout(cmd *cobra.Command, timeoutFlag string) (time.Duration, error) {
	// Get the wait value from the command line
	wait, err := cmd.PersistentFlags().GetBool(constants.WaitFlag)
	if err != nil {
		return time.Duration(0), err
	}
	if wait {
		timeout, err := cmd.PersistentFlags().GetDuration(timeoutFlag)
		if err != nil {
			return time.Duration(0), err
		}
		return timeout, nil
	}

	// Return duration of zero since --wait=false was specified
	return time.Duration(0), nil
}

// GetLogFormat returns the format type for streaming log output
func GetLogFormat(cmd *cobra.Command) (LogFormat, error) {
	// Get the log format value from the command line
	logFormat := cmd.PersistentFlags().Lookup(constants.LogFormatFlag)
	if logFormat == nil {
		return LogFormatSimple, nil
	}

	return LogFormat(logFormat.Value.String()), nil
}

// GetVersion returns the version of Verrazzano to install/upgrade
func GetVersion(cmd *cobra.Command, vzHelper helpers.VZHelper) (string, error) {
	// Get the version from the command line
	version, err := cmd.PersistentFlags().GetString(constants.VersionFlag)
	if err != nil {
		return "", err
	}

	// If the user has provided an operator YAML, attempt to get the version from the VPO deployment
	if ManifestsFlagChanged(cmd) {
		manifestsVersion, err := getVersionFromOperatorYAML(cmd, vzHelper)
		if err != nil {
			return "", err
		}

		if manifestsVersion != "" {
			// If the user has explicitly passed a version, make sure it matches the version in the manifests
			if cmd.PersistentFlags().Changed(constants.VersionFlag) {
				match, err := versionsMatch(manifestsVersion, version)
				if err != nil {
					return "", err
				}
				if match {
					// Return version and not manifestsVersion because version may have prerelease and build values that are
					// not present in the manifests version, and make sure it has a "v" prefix
					if !strings.HasPrefix(version, "v") {
						version = "v" + version
					}
					return version, nil
				}
				return "", fmt.Errorf("Requested version '%s' does not match manifests version '%s'", version, manifestsVersion)
			}

			return manifestsVersion, nil
		}
	}

	if version == constants.VersionFlagDefault {
		// Find the latest release version of Verrazzano
		version, err = helpers.GetLatestReleaseVersion(vzHelper.GetHTTPClient())
		if err != nil {
			return "", err
		}
	} else {
		// Validate the version string
		installVersion, err := semver.NewSemVersion(version)
		if err != nil {
			return "", err
		}
		version = fmt.Sprintf("v%s", installVersion.ToString())
	}
	return version, nil
}

// getVersionFromOperatorYAML attempts to parse the user-provided operator YAML and returns the
// Verrazzano version from a label on the verrazzano-platform-operator deployment.
func getVersionFromOperatorYAML(cmd *cobra.Command, vzHelper helpers.VZHelper) (string, error) {
	localOperatorFilename, userVisibleFilename, isTempFile, err := getOrDownloadOperatorYAML(cmd, "", vzHelper)
	if err != nil {
		return "", err
	}
	if isTempFile {
		// the operator YAML is a temporary file that must be deleted after applying it
		defer os.Remove(localOperatorFilename)
	}

	fileObj, err := os.Open(localOperatorFilename)
	defer func() { fileObj.Close() }()
	if err != nil {
		return "", err
	}
	objectsInYAML, err := k8sutil.Unmarshall(bufio.NewReader(fileObj))
	if err != nil {
		return "", err
	}
	vpoDeployIdx, _ := findVPODeploymentIndices(objectsInYAML)
	if vpoDeployIdx == -1 {
		return "", fmt.Errorf("Unable to find verrazzano-platform-operator deployment in operator file: %s", userVisibleFilename)
	}

	vpoDeploy := &objectsInYAML[vpoDeployIdx]
	version, found, err := unstructured.NestedString(vpoDeploy.Object, "metadata", "labels", "app.kubernetes.io/version")
	if err != nil || !found {
		return "", err
	}

	// versions we return are expected to start with "v"
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return version, err
}

// versionsMatch returns true if the versions are semantically equivalent. Only the major, minor, and patch fields are considered.
func versionsMatch(left, right string) (bool, error) {
	leftVersion, err := semver.NewSemVersion(left)
	if err != nil {
		return false, err
	}
	rightVersion, err := semver.NewSemVersion(right)
	if err != nil {
		return false, err
	}

	// When comparing the versions, ignore the prerelease and build versions. This is needed to support development scenarios
	// where the version to upgrade to looks like x.y.z-nnnn+hash but the VPO version label is x.y.z.
	leftVersion.Prerelease = ""
	leftVersion.Build = ""
	rightVersion.Prerelease = ""
	rightVersion.Build = ""

	return leftVersion.IsEqualTo(rightVersion), nil
}

// ConfirmWithUser asks the user a yes/no question and returns true if the user answered yes, false
// otherwise.
func ConfirmWithUser(vzHelper helpers.VZHelper, questionText string, skipQuestion bool) (bool, error) {
	if skipQuestion {
		return true, nil
	}
	var response string
	scanner := bufio.NewScanner(vzHelper.GetInputStream())
	fmt.Fprintf(vzHelper.GetOutputStream(), "%s [y/N]: ", questionText)
	if scanner.Scan() {
		response = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if response == "y" || response == "Y" {
		return true, nil
	}
	return false, nil
}

// getOperatorFileFromFlag returns the value for the manifests (or the alias operator-file) option
func getOperatorFileFromFlag(cmd *cobra.Command) (string, error) {
	// Get the value from the command line
	operatorFile, err := getManifestsFile(cmd)
	if err != nil {
		return "", fmt.Errorf("Failed to parse the command line option %s: %s", constants.ManifestsFlag, err.Error())
	}
	return operatorFile, nil
}

// getManifestsFile returns the manifests file, which could come from the manifests flag or the
// deprecated operator-file flag
func getManifestsFile(cmd *cobra.Command) (string, error) {
	// if manifests flag has been explicitly provided, use that. Else if operator-file flag is
	// explicitly provided, use that. If neither is explicitly provided, use the default for the
	// manifests flag
	if cmd.PersistentFlags().Changed(constants.ManifestsFlag) {
		return cmd.PersistentFlags().GetString(constants.ManifestsFlag)
	}
	if cmd.PersistentFlags().Changed(constants.OperatorFileFlag) {
		return cmd.PersistentFlags().GetString(constants.OperatorFileFlag)
	}
	// neither is explicitly specified, use the default value of manifests flag
	return cmd.PersistentFlags().GetString(constants.ManifestsFlag)
}

// ManifestsFlagChanged returns whether the manifests flag (or deprecated operator-file flag) is specified.
func ManifestsFlagChanged(cmd *cobra.Command) bool {
	return cmd.PersistentFlags().Changed(constants.ManifestsFlag) || cmd.PersistentFlags().Changed(constants.OperatorFileFlag)
}

// AddManifestsFlags adds flags related to providing manifests (including the deprecated
// operator-file flag as an alias for the manifests flag)
func AddManifestsFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP(constants.ManifestsFlag, constants.ManifestsShorthand, "", constants.ManifestsFlagHelp)
	// The operator-file flag is left in as an alias for the manifests flag
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.ManifestsFlagHelp)
	cmd.PersistentFlags().MarkDeprecated(constants.OperatorFileFlag, constants.OperatorFileDeprecateMsg)
}

// GetSetArguments gets all the set arguments and returns a map of property/value
func GetSetArguments(cmd *cobra.Command, vzHelper helpers.VZHelper) (map[string]string, error) {
	setMap := make(map[string]string)
	setFlags, err := cmd.PersistentFlags().GetStringArray(constants.SetFlag)
	if err != nil {
		return nil, err
	}

	invalidFlag := false
	for _, setFlag := range setFlags {
		pv := strings.Split(setFlag, "=")
		if len(pv) != 2 {
			fmt.Fprintf(vzHelper.GetErrorStream(), fmt.Sprintf("Invalid set flag \"%s\" specified. Flag must be specified in the format path=value\n", setFlag))
			invalidFlag = true
			continue
		}
		if !invalidFlag {
			path, value := strings.TrimSpace(pv[0]), strings.TrimSpace(pv[1])
			if !strings.HasPrefix(path, "spec.") {
				path = "spec." + path
			}
			setMap[path] = value
		}
	}

	if invalidFlag {
		return nil, fmt.Errorf("Invalid set flag(s) specified")
	}

	return setMap, nil
}

// GenerateYAMLForSetFlags creates a YAML string from a map of property value pairs representing --set flags
// specified on the install command
func GenerateYAMLForSetFlags(pvs map[string]string) (string, error) {
	yamlObject := map[string]interface{}{}
	for path, value := range pvs {
		// replace unwanted characters in the value to avoid splitting
		ignoreChars := ",[.{}"
		for _, char := range ignoreChars {
			value = strings.Replace(value, string(char), "\\"+string(char), -1)
		}

		composedStr := fmt.Sprintf("%s=%s", path, value)
		err := strvals.ParseInto(composedStr, yamlObject)
		if err != nil {
			return "", err
		}
	}

	yamlFile, err := yaml.Marshal(yamlObject)
	if err != nil {
		return "", err
	}

	yamlString := string(yamlFile)

	// Replace any double-quoted strings that are surrounded by single quotes.
	// These type of strings are problematic for helm.
	yamlString = strings.ReplaceAll(yamlString, "'\"", "\"")
	yamlString = strings.ReplaceAll(yamlString, "\"'", "\"")

	return yamlString, nil
}
