// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/kubectlutil"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/bugreport"
	"time"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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

	// Initially the operator-file flag may be for internal use, hide from help until
	// a decision is made on supporting this option.
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.OperatorFileFlagHelp)
	cmd.PersistentFlags().MarkHidden(constants.OperatorFileFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an upgrade.")
	cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	// Hide the flag for overriding the default wait timeout for the platform-operator
	cmd.PersistentFlags().MarkHidden(constants.VPOTimeoutFlag)

	return cmd
}

func runCmdUpgrade(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
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
		// Delete leftover verrazzano-operator deployment after an abort.
		// This allows for the verrazzano-operator validatingWebhookConfiguration to be updated with the correct caBundle.
		err = cmdhelpers.DeleteFunc(client)
		if err != nil {
			return err
		}

		// Apply the Verrazzano operator.yaml
		err = cmdhelpers.ApplyPlatformOperatorYaml(cmd, client, vzHelper, version)
		if err != nil {
			return err
		}

		err = upgradeVerrazzano(vzHelper, vz, client, version, vpoTimeout)
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
	// install resource. This could happen if the upgrade command was aborted and the rerun. We anly wait for the upgrade
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

func upgradeVerrazzano(vzHelper helpers.VZHelper, vz *v1beta1.Verrazzano, client clipkg.Client, version string, vpoTimeout time.Duration) error {
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
			err = helpers.UpdateVerrazzanoResource(client, vz)
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
