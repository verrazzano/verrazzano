// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/kubectlutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/bugreport"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kube-openapi/pkg/util/proto/validation"
	"k8s.io/kubectl/pkg/util/openapi"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

const (
	CommandName = "install"
	helpShort   = "Install Verrazzano"
	helpLong    = `Install the Verrazzano Platform Operator and install the Verrazzano components specified by the Verrazzano CR provided on the command line`
)

var helpExample = fmt.Sprintf(`
# Install the latest version of Verrazzano using the prod profile. Stream the logs to the console until the install completes.
vz install

# Install version %[1]s using a dev profile, timeout the command after 20 minutes.
vz install --version v%[1]s --set profile=dev --timeout 20m

# Install version %[1]s using a dev profile with kiali disabled and wait for the install to complete.
vz install --version v%[1]s --set profile=dev --set components.kiali.enabled=false

# Install the latest version of Verrazzano using CR overlays and explicit value sets.  Output the logs in json format.
# The overlay files can be a comma-separated list or a series of -f options.  Both formats are shown.
vz install -f base.yaml,custom.yaml --set profile=prod --log-format json
vz install -f base.yaml -f custom.yaml --set profile=prod --log-format json

# Install the latest version of Verrazzano using a Verrazzano CR specified with stdin.
vz install -f - <<EOF
apiVersion: install.verrazzano.io/v1beta1
kind: Verrazzano
metadata:
  namespace: default
  name: example-verrazzano
EOF`, version.GetCLIVersion())

var logsEnum = cmdhelpers.LogFormatSimple

// validateCR functions used for unit-tests
type validateCRSig func(cmd *cobra.Command, obj *unstructured.Unstructured, vzHelper helpers.VZHelper) []error

var ValidateCRFunc validateCRSig = ValidateCR

func SetValidateCRFunc(f validateCRSig) {
	ValidateCRFunc = f
}

func SetDefaultValidateCRFunc() {
	ValidateCRFunc = ValidateCR
}

func FakeValidateCRFunc(cmd *cobra.Command, obj *unstructured.Unstructured, vzHelper helpers.VZHelper) []error {
	return nil
}

func NewCmdInstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdInstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().Duration(constants.VPOTimeoutFlag, time.Minute*5, constants.VPOTimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, constants.VersionFlagInstallHelp)
	cmd.PersistentFlags().StringSliceP(constants.FilenameFlag, constants.FilenameFlagShorthand, []string{}, constants.FilenameFlagHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)
	cmd.PersistentFlags().StringArrayP(constants.SetFlag, constants.SetFlagShorthand, []string{}, constants.SetFlagHelp)
	cmd.PersistentFlags().Bool(constants.AutoBugReportFlag, constants.AutoBugReportFlagDefault, constants.AutoBugReportFlagHelp)
	// Private registry support
	cmd.PersistentFlags().String(constants.ImageRegistryFlag, constants.ImageRegistryFlagDefault, constants.ImageRegistryFlagHelp)
	cmd.PersistentFlags().String(constants.ImagePrefixFlag, constants.ImagePrefixFlagDefault, constants.ImagePrefixFlagHelp)

	// Flag to skip any confirmation questions
	cmd.PersistentFlags().BoolP(constants.SkipConfirmationFlag, constants.SkipConfirmationShort, false, constants.SkipConfirmationFlagHelp)
	// Flag to skip reinstalling the operator
	cmd.PersistentFlags().BoolP(constants.SkipOperatorInstallFlag, constants.SkipOperatorInstallShort, false, constants.SkipOperatorInstallFlagHelp)
	// Add flags related to specifying the platform operator manifests as a local file or a URL
	cmdhelpers.AddManifestsFlags(cmd)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an install.")
	cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	// Hide the flag for overriding the default wait timeout for the platform-operator
	cmd.PersistentFlags().MarkHidden(constants.VPOTimeoutFlag)

	// Verifies that the CLI args are not set at the creation of a command
	vzHelper.VerifyCLIArgsNil(cmd)

	return cmd
}

func runCmdInstall(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	// Validate the command options
	err := validateCmd(cmd)
	if err != nil {
		return fmt.Errorf("Command validation failed: %s", err.Error())
	}

	// Get the timeout value for the install command
	timeout, err := cmdhelpers.GetWaitTimeout(cmd, constants.TimeoutFlag)
	if err != nil {
		return err
	}

	// Get the log format value
	logFormat, err := cmdhelpers.GetLogFormat(cmd)
	if err != nil {
		return err
	}

	// Get the kubernetes clientset.  This will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Get the controller runtime client
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// When manifests flag is not used, get the version from the command line
	var version string
	if !cmdhelpers.ManifestsFlagChanged(cmd) {
		version, err = cmdhelpers.GetVersion(cmd, vzHelper)
		if err != nil {
			return err
		}
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Installing Verrazzano version %s\n", version))
	}

	var vzNamespace string
	var vzName string

	// Get the VPO timeout
	vpoTimeout, err := cmdhelpers.GetWaitTimeout(cmd, constants.VPOTimeoutFlag)
	if err != nil {
		return err
	}

	// Check to see if we have a vz resource already deployed
	existingvz, _ := helpers.FindVerrazzanoResource(client)
	if existingvz != nil {
		// Allow install command to continue if an install is in progress and the same version is specified.
		// For example, control-C was entered and the install command is run again.
		// Note: "Installing" is a state that was used in pre 1.4.0 installs and replaced with "Reconciling".
		if existingvz.Status.State != v1beta1.VzStateReconciling && existingvz.Status.State != "Installing" {
			return fmt.Errorf("Only one install of Verrazzano is allowed")
		}

		if version != "" {
			installVersion, err := semver.NewSemVersion(version)
			if err != nil {
				return fmt.Errorf("Failed creating semantic version from install version %s: %s", version, err.Error())
			}
			vzVersion, err := semver.NewSemVersion(existingvz.Status.Version)
			if err != nil {
				return fmt.Errorf("Failed creating semantic version from Verrazzano status version %s: %s", existingvz.Status.Version, err.Error())
			}
			if !installVersion.IsEqualTo(vzVersion) {
				return fmt.Errorf("Unable to install version %s, install of version %s is in progress", version, existingvz.Status.Version)
			}
		}

		if err := cmdhelpers.ValidatePrivateRegistry(cmd, client); err != nil {
			skipConfirm, errConfirm := cmd.PersistentFlags().GetBool(constants.SkipConfirmationFlag)
			if errConfirm != nil {
				return errConfirm
			}
			proceed, err := cmdhelpers.ConfirmWithUser(vzHelper, fmt.Sprintf("%s\nYour new settings will be ignored. Continue?", err.Error()), skipConfirm)
			if err != nil {
				return err
			}
			if !proceed {
				fmt.Fprintf(vzHelper.GetOutputStream(), "Operation canceled.")
				return nil
			}
		}
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Install of Verrazzano version %s is already in progress\n", version))

		vzNamespace = existingvz.Namespace
		vzName = existingvz.Name
	} else {
		// Get the verrazzano install resource to be created
		vz, obj, err := getVerrazzanoYAML(cmd, vzHelper, version)
		if err != nil {
			return err
		}

		// Delete leftover verrazzano-platform-operator deployments after an abort.
		// This allows for the verrazzano-platform-operator validatingWebhookConfiguration to be updated with the correct caBundle.
		err = cmdhelpers.DeleteFunc(client)
		if err != nil {
			return err
		}

		if !cmd.PersistentFlags().Changed(constants.SkipOperatorInstallFlag) {
			// Apply the Verrazzano operator.yaml.
			err = cmdhelpers.ApplyPlatformOperatorYaml(cmd, client, vzHelper, version)
			if err != nil {
				return err
			}
		}

		err = installVerrazzano(cmd, vzHelper, vz, client, version, vpoTimeout, obj)
		if err != nil {
			return bugreport.AutoBugReport(cmd, vzHelper, err)
		}
		vzNamespace = vz.GetNamespace()
		vzName = vz.GetName()
	}

	// Wait for the Verrazzano install to complete
	err = waitForInstallToComplete(client, kubeClient, vzHelper, types.NamespacedName{Namespace: vzNamespace, Name: vzName}, timeout, vpoTimeout, logFormat)
	if err != nil {
		return bugreport.AutoBugReport(cmd, vzHelper, err)
	}
	return nil
}

func installVerrazzano(cmd *cobra.Command, vzHelper helpers.VZHelper, vz clipkg.Object, client clipkg.Client, version string, vpoTimeout time.Duration, obj *unstructured.Unstructured) error {
	// Wait for the platform operator to be ready before we create the Verrazzano resource.
	_, err := cmdhelpers.WaitForPlatformOperator(client, vzHelper, v1beta1.CondInstallComplete, vpoTimeout)
	if err != nil {
		return err
	}

	// Validate Custom Resource if present
	var errorArray = ValidateCRFunc(cmd, obj, vzHelper)
	if len(errorArray) != 0 {
		return fmt.Errorf("was unable to validate the given CR, the following error(s) occurred: \"%v\"", errorArray)
	}

	err = kubectlutil.SetLastAppliedConfigurationAnnotation(vz)
	if err != nil {
		return err
	}

	// Create the Verrazzano install resource, if need be.
	// We will retry up to 5 times if there is an error.
	// Sometimes we see intermittent webhook errors due to timeouts.
	retry := 0
	for {
		err = client.Create(context.TODO(), vz)
		if err != nil {
			if retry == 5 {
				return fmt.Errorf("Failed to create the verrazzano install resource: %s", err.Error())
			}
			time.Sleep(time.Second)
			retry++
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Retrying after failing to create the verrazzano install resource: %s\n", err.Error()))
			continue
		}
		break
	}
	return nil
}

// getVerrazzanoYAML returns the verrazzano install resource to be created
func getVerrazzanoYAML(cmd *cobra.Command, vzHelper helpers.VZHelper, version string) (vz clipkg.Object, obj *unstructured.Unstructured, err error) {
	// Get the list yaml filenames specified
	filenames, err := cmd.PersistentFlags().GetStringSlice(constants.FilenameFlag)
	if err != nil {
		return nil, nil, err
	}

	// Get the set arguments - returning a list of properties and value
	pvs, err := getSetArguments(cmd, vzHelper)
	if err != nil {
		return nil, nil, err
	}

	// If no yamls files were passed on the command line then create a minimal verrazzano
	// resource.  The minimal resource is used to create a resource called verrazzano
	// in the default namespace using the prod profile.
	var gv schema.GroupVersion
	if len(filenames) == 0 {
		gv, vz, err = helpers.NewVerrazzanoForVZVersion(version)
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Merge the yaml files passed on the command line
		obj, err := cmdhelpers.MergeYAMLFiles(filenames, os.Stdin)
		if err != nil {
			return nil, nil, err
		}
		gv = obj.GroupVersionKind().GroupVersion()
		vz = obj
	}

	// Generate yaml for the set flags passed on the command line
	outYAML, err := generateYAMLForSetFlags(pvs)
	if err != nil {
		return nil, nil, err
	}

	// Merge the set flags passed on the command line. The set flags take precedence over
	// the yaml files passed on the command line.
	vz, unstructuredVZObj, err := cmdhelpers.MergeSetFlags(gv, vz, outYAML)
	if err != nil {
		return nil, nil, err
	}

	// Return the merged verrazzano install resource to be created
	return vz, unstructuredVZObj, nil
}

// generateYAMLForSetFlags creates a YAML string from a map of property value pairs representing --set flags
// specified on the install command
func generateYAMLForSetFlags(pvs map[string]string) (string, error) {
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

// getSetArguments gets all the set arguments and returns a map of property/value
func getSetArguments(cmd *cobra.Command, vzHelper helpers.VZHelper) (map[string]string, error) {
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

// waitForInstallToComplete waits for the Verrazzano install to complete and shows the logs of
// the ongoing Verrazzano install.
func waitForInstallToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, namespacedName types.NamespacedName, timeout time.Duration, vpoTimeout time.Duration, logFormat cmdhelpers.LogFormat) error {
	return cmdhelpers.WaitForOperationToComplete(client, kubeClient, vzHelper, namespacedName, timeout, vpoTimeout, logFormat, v1beta1.CondInstallComplete)
}

// validateCmd - validate the command line options
func validateCmd(cmd *cobra.Command) error {
	if cmd.PersistentFlags().Changed(constants.VersionFlag) && cmdhelpers.ManifestsFlagChanged(cmd) {
		return fmt.Errorf("--%s and --%s cannot both be specified", constants.VersionFlag, constants.ManifestsFlag)
	}
	if cmd.PersistentFlags().Changed(constants.SkipOperatorInstallFlag) && cmdhelpers.ManifestsFlagChanged(cmd) {
		return fmt.Errorf("--%s and --%s cannot both be specified", constants.SkipOperatorInstallFlag, constants.ManifestsFlag)
	}
	prefix, err := cmd.PersistentFlags().GetString(constants.ImagePrefixFlag)
	if err != nil {
		return err
	}
	reg, err := cmd.PersistentFlags().GetString(constants.ImageRegistryFlag)
	if err != nil {
		return err
	}
	if prefix != constants.ImagePrefixFlagDefault && reg == constants.ImageRegistryFlagDefault {
		return fmt.Errorf("%s cannot be specified without also specifying %s", constants.ImagePrefixFlag, constants.ImageRegistryFlag)
	}
	return nil
}

// validateCR - validates a Custom Resource before proceeding with an install
func ValidateCR(cmd *cobra.Command, obj *unstructured.Unstructured, vzHelper helpers.VZHelper) []error {
	discoveryClient, err := vzHelper.GetDiscoveryClient(cmd)
	if err != nil {
		return []error{err}
	}
	doc, err := discoveryClient.OpenAPISchema()
	if err != nil {
		return []error{err}
	}
	s, err := openapi.NewOpenAPIData(doc)
	if err != nil {
		return []error{err}
	}

	gvk := obj.GroupVersionKind()
	schema := s.LookupResource(gvk)
	if schema == nil {
		return []error{fmt.Errorf("the schema for \"%v\" was not found", gvk.Kind)}
	}

	// ValidateModel validates a given schema
	errorArray := validation.ValidateModel(obj.Object, schema, gvk.Kind)
	if len(errorArray) != 0 {
		return errorArray
	}

	return nil
}
