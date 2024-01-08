// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"time"
)

const (
	CommandName = "upgrade"
	helpShort   = "Upgrade Verrazzano"
	helpLong    = `Upgrade the Verrazzano Platform Operator to the specified version and update all of the currently installed components`
)

var helpExample = fmt.Sprintf(`
# Upgrade to the latest version of Verrazzano and wait for the command to complete.  Stream the logs to the console until the upgrade completes.
vz upgrade

# Upgrade to Verrazzano v%[1]s, stream the logs to the console and timeout after 20m
vz upgrade --version v%[1]s --timeout 20m`, version.GetCLIVersion())

var logsEnum = cmdhelpers.LogFormatSimple

func NewCmdUpgrade(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUpgrade(cmd, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().Duration(constants.VPOTimeoutFlag, time.Minute*5, constants.VPOTimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, constants.VersionFlagUpgradeHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)
	cmd.PersistentFlags().Bool(constants.AutoBugReportFlag, constants.AutoBugReportFlagDefault, constants.AutoBugReportFlagHelp)
	// Private registry support
	cmd.PersistentFlags().String(constants.ImageRegistryFlag, constants.ImageRegistryFlagDefault, constants.ImageRegistryFlagHelp)
	cmd.PersistentFlags().String(constants.ImagePrefixFlag, constants.ImagePrefixFlagDefault, constants.ImagePrefixFlagHelp)

	// Add flags related to specifying the platform operator manifests as a local file or a URL
	cmdhelpers.AddManifestsFlags(cmd)

	// Flag to skip any confirmation questions
	cmd.PersistentFlags().BoolP(constants.SkipConfirmationFlag, constants.SkipConfirmationShort, false, constants.SkipConfirmationFlagHelp)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an upgrade.")
	cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	// Hide the flag for overriding the default wait timeout for the platform-operator
	cmd.PersistentFlags().MarkHidden(constants.VPOTimeoutFlag)

	// Set Flag for setting args
	cmd.PersistentFlags().StringArrayP(constants.SetFlag, constants.SetFlagShorthand, []string{}, constants.SetFlagHelp)

	// Verifies that the CLI args are not set at the creation of a command
	vzHelper.VerifyCLIArgsNil(cmd)

	return cmd
}

func runCmdUpgrade(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	fmt.Fprintf(vzHelper.GetOutputStream(), version.GetVZCLIVersionMessageString())
	if err := validateCmd(cmd); err != nil {
		return fmt.Errorf("Command validation failed: %s", err.Error())
	}

	// Get the controller runtime client
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Find the verrazzano resource that needs to be updated to the new version
	vz, err := helpers.FindVerrazzanoResource(client)
	if err != nil {
		return fmt.Errorf("Verrazzano is not installed: %s", err.Error())
	}

	skipConfirm, errConfirm := cmd.PersistentFlags().GetBool(constants.SkipConfirmationFlag)
	if errConfirm != nil {
		return errConfirm
	}

	// If the Verrazzano resource is already Reconciling, confirm with user that they would like to
	// proceed with an upgrade
	if vz.Status.State == v1beta1.VzStateReconciling {
		proceed, err := cmdhelpers.ConfirmWithUser(vzHelper, "Verrazzano is already in the middle of an install. Continue with upgrade anyway?", skipConfirm)
		if err != nil {
			return err
		}
		if !proceed {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Operation canceled.")
			return nil
		}
	}

	// Validate any existing private registry settings against new ones and get confirmation from the user
	if err := cmdhelpers.ValidatePrivateRegistry(cmd, client); err != nil {
		proceed, err := cmdhelpers.ConfirmWithUser(vzHelper, fmt.Sprintf("%s\nProceed to upgrade with new settings?", err.Error()), skipConfirm)
		if err != nil {
			return err
		}
		if !proceed {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Upgrade canceled.")
			return nil
		}
	}

	// Get the version Verrazzano is being upgraded to
	version, err := cmdhelpers.GetVersion(cmd, vzHelper)
	if err != nil {
		return err
	}

	vzStatusVersion, err := semver.NewSemVersion(vz.Status.Version)
	if err != nil {
		return fmt.Errorf("Failed creating semantic version from Verrazzano status version %s: %s", vz.Status.Version, err.Error())
	}
	upgradeVersion, err := semver.NewSemVersion(version)
	if err != nil {
		return fmt.Errorf("Failed creating semantic version from version %s specified: %s", version, err.Error())
	}

	// Version being upgraded to cannot be less than the installed version
	if upgradeVersion.IsLessThan(vzStatusVersion) {
		return fmt.Errorf("Upgrade to a lesser version of Verrazzano is not allowed. Upgrade version specified was %s and current Verrazzano version is %s", version, vz.Status.Version)
	}

	var vzSpecVersion *semver.SemVersion
	if vz.Spec.Version != "" {
		vzSpecVersion, err = semver.NewSemVersion(vz.Spec.Version)
		if err != nil {
			return fmt.Errorf("Failed creating semantic version from Verrazzano spec version %s: %s", vz.Spec.Version, err.Error())
		}
		// Version being upgraded to cannot be less than version previously specified during an upgrade
		if upgradeVersion.IsLessThan(vzSpecVersion) {
			return fmt.Errorf("Upgrade to a lesser version of Verrazzano is not allowed. Upgrade version specified was %s and the upgrade in progress is %s", version, vz.Spec.Version)
		}
	} else {
		// Installed version is already at the upgrade version specified
		if upgradeVersion.IsEqualTo(vzStatusVersion) {
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Verrazzano is already at the specified upgrade version of %s\n", version))
			return nil
		}
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Upgrading Verrazzano to version %s\n", version))

	// Get the timeout value for the upgrade command
	timeout, err := cmdhelpers.GetWaitTimeout(cmd, constants.TimeoutFlag)
	if err != nil {
		return err
	}

	// Get the log format value
	logFormat, err := cmdhelpers.GetLogFormat(cmd)
	if err != nil {
		return err
	}

	// Get the VPO timeout
	vpoTimeout, err := cmdhelpers.GetWaitTimeout(cmd, constants.VPOTimeoutFlag)
	if err != nil {
		return err
	}

	// Get the kubernetes clientset.  This will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	if vz.Spec.Version == "" || !upgradeVersion.IsEqualTo(vzSpecVersion) {

		// Apply the Verrazzano operator.yaml
		err = cmdhelpers.ApplyPlatformOperatorYaml(cmd, client, vzHelper, version)
		if err != nil {
			return err
		}

		err = upgradeVerrazzano(cmd, vzHelper, vz, client, version, vpoTimeout)
		if err != nil {
			return bugreport.AutoBugReport(cmd, vzHelper, err)
		}

		// Wait for the Verrazzano upgrade to complete
		err = waitForUpgradeToComplete(client, kubeClient, vzHelper, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, timeout, vpoTimeout, logFormat)
		if err != nil {
			return bugreport.AutoBugReport(cmd, vzHelper, err)
		}
		return nil
	}

	// If we already started the upgrade no need to apply the operator.yaml, wait for VPO, and update the verrazzano
	// install resource. This could happen if the upgrade command was aborted and then rerun. We only wait for the upgrade
	// to complete.
	if !vzStatusVersion.IsEqualTo(vzSpecVersion) {
		err = waitForUpgradeToComplete(client, kubeClient, vzHelper, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, timeout, vpoTimeout, logFormat)
		if err != nil {
			return bugreport.AutoBugReport(cmd, vzHelper, err)
		}
		return nil
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Verrazzano has already been upgraded to version %s\n", vz.Status.Version))
	return nil
}

func upgradeVerrazzano(cmd *cobra.Command, vzHelper helpers.VZHelper, vz *v1beta1.Verrazzano, client clipkg.Client, version string, vpoTimeout time.Duration) error {
	// Wait for the platform operator to be ready before we update the verrazzano install resource
	_, err := cmdhelpers.WaitForPlatformOperator(client, vzHelper, v1beta1.CondUpgradeComplete, vpoTimeout)
	if err != nil {
		return err
	}

	err = kubectlutil.SetLastAppliedConfigurationAnnotation(vz)
	if err != nil {
		return err
	}

	// Update the version in the verrazzano install resource.  This will initiate the Verrazzano upgrade.
	// We will retry up to 5 times if there is an error.
	// Sometimes we see intermittent webhook errors due to timeouts.
	retry := 0
	for {
		// Get the verrazzano install resource each iteration, in case of resource conflicts
		vz, err = helpers.GetVerrazzanoResource(client, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name})
		if err == nil {
			vz.Spec.Version = version

			vzWithSetFlags, err := mergeSetFlagsIntoVerrazzanoResource(cmd, vzHelper, vz)
			if err != nil {
				return err
			}
			if vzWithSetFlags != nil {
				vz = vzWithSetFlags
			}

			err = helpers.UpdateVerrazzanoResource(client, vz)
			if err != nil {
				return err
			}
		}
		if err != nil {
			if retry == 5 {
				return fmt.Errorf("Failed to set the upgrade version in the verrazzano install resource: %s", err.Error())
			}
			time.Sleep(time.Second)
			retry++
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Retrying after failing to set the upgrade version in the verrazzano install resource: %s\n", err.Error()))
			continue
		}
		break
	}
	return nil
}

// Wait for the upgrade operation to complete
func waitForUpgradeToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, namespacedName types.NamespacedName, timeout time.Duration, vpoTimeout time.Duration, logFormat cmdhelpers.LogFormat) error {
	return cmdhelpers.WaitForOperationToComplete(client, kubeClient, vzHelper, namespacedName, timeout, vpoTimeout, logFormat, v1beta1.CondUpgradeComplete)
}

// validateCmd - validate the command line options
func validateCmd(cmd *cobra.Command) error {
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

// mergeSetFlagsIntoVerrazzanoResource - determines if any --set flags are used and merges them into the existing Verrazzano resource to be applied at upgrade
func mergeSetFlagsIntoVerrazzanoResource(cmd *cobra.Command, vzHelper helpers.VZHelper, vz *v1beta1.Verrazzano) (*v1beta1.Verrazzano, error) {
	// Check that set flags are set. Otherwise, nothing is returned and the Verrazzano resource is left untouched
	setFlags, _ := cmd.PersistentFlags().GetStringArray(constants.SetFlag)
	if len(setFlags) != 0 {
		// Get the set arguments - returning a list of properties and value
		pvs, err := cmdhelpers.GetSetArguments(cmd, vzHelper)
		if err != nil {
			return nil, err
		}

		// Generate yaml for the set flags passed on the command line
		outYAML, err := cmdhelpers.GenerateYAMLForSetFlags(pvs)
		if err != nil {
			return nil, err
		}

		// Merge the set flags passed on the command line. The set flags take precedence over
		// the yaml files passed on the command line.
		mergedVZ, _, err := cmdhelpers.MergeSetFlagsUpgrade(vz.GroupVersionKind().GroupVersion(), vz, outYAML)
		if err != nil {
			return nil, err
		}

		// Update requires a Verrazzano resource to apply changes, so here the client Object becomes a Verrazzano resource to be returned
		vzMarshalled, err := yaml.Marshal(mergedVZ)
		if err != nil {
			return nil, err
		}
		newVZ := v1beta1.Verrazzano{}
		err = yaml.Unmarshal(vzMarshalled, &newVZ)
		if err != nil {
			return nil, err
		}

		return &newVZ, err
	}
	return nil, nil
}
