// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CommandName = "install"
	helpShort   = "Install Verrazzano"
	helpLong    = `Install the Verrazzano Platform Operator and install the Verrazzano components specified by the Verrazzano CR provided on the command line`
	helpExample = `
# Install the latest version of Verrazzano using the prod profile. Stream the logs to the console until the install completes.
vz install

# Install version 1.3.0 using a dev profile, timeout the command after 20 minutes
vz install --version v1.3.0 --set profile=dev --timeout 20m

# Install version 1.3.0 using a dev profile with elasticsearch disabled and wait for the install to complete
vz install --version v1.3.0 --set profile=dev --set components.elasticsearch.enabled=false

# Install the latest version of Verrazzano using CR overlays and explicit value sets.  Output the logs in json format.
vz install -f base.yaml -f custom.yaml --set profile=prod --log-format json`
)

var logsEnum = cmdhelpers.LogFormatSimple

func NewCmdInstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdInstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, constants.VersionFlagInstallHelp)
	cmd.PersistentFlags().StringSliceP(constants.FilenameFlag, constants.FilenameFlagShorthand, []string{}, constants.FilenameFlagHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)
	cmd.PersistentFlags().StringArrayP(constants.SetFlag, constants.SetFlagShorthand, []string{}, constants.SetFlagHelp)

	// Initially the operator-file flag may be for internal use, hide from help until
	// a decision is made on supporting this option.
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.OperatorFileFlagHelp)
	cmd.PersistentFlags().MarkHidden(constants.OperatorFileFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an install.")
	cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	return cmd
}

func runCmdInstall(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	// Validate the command options
	err := cmdhelpers.ValidateCmd(cmd)
	if err != nil {
		return fmt.Errorf("Command validation failed: %s", err.Error())
	}

	// Get the verrazzano install resource to be created
	vz, err := getVerrazzanoYAML(cmd)
	if err != nil {
		return err
	}

	// Get the timeout value for the install command
	timeout, err := cmdhelpers.GetWaitTimeout(cmd)
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

	// When --operator-file is not used, get the version from the command line
	var version string
	if !cmd.PersistentFlags().Changed(constants.OperatorFileFlag) {
		version, err = cmdhelpers.GetVersion(cmd, vzHelper)
		if err != nil {
			return err
		}
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Installing Verrazzano version %s\n", version))
	}

	// Apply the Verrazzano operator.yaml.
	err = cmdhelpers.ApplyPlatformOperatorYaml(cmd, client, vzHelper, version)
	if err != nil {
		return err
	}

	// Wait for the platform operator to be ready before we create the Verrazzano resource.
	vpoPodName, err := cmdhelpers.WaitForPlatformOperator(client, vzHelper, vzapi.CondInstallComplete)
	if err != nil {
		return err
	}

	// Create the Verrazzano install resource.
	err = client.Create(context.TODO(), vz)
	if err != nil {
		return fmt.Errorf("Failed to create verrazzano resource: %s", err.Error())
	}

	// Wait for the Verrazzano install to complete
	return waitForInstallToComplete(client, kubeClient, vzHelper, vpoPodName, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, timeout, logFormat)
}

// getVerrazzanoYAML returns the verrazzano install resource to be created
func getVerrazzanoYAML(cmd *cobra.Command) (vz *vzapi.Verrazzano, err error) {
	filenames, err := cmd.PersistentFlags().GetStringSlice(constants.FilenameFlag)
	if err != nil {
		return nil, err
	}

	// If no yamls files were passed on the command line then return a minimal verrazzano
	// resource.  The minimal resource will be used to create a resource called verrazzano
	// in the default namespace using the prod profile.
	if len(filenames) == 0 {
		vz = &vzapi.Verrazzano{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "verrazzano",
			},
		}
		return vz, nil
	}

	// Merge the yaml files passed on the command line and return the merged verrazzano resource
	// to be created.
	return cmdhelpers.MergeYAMLFiles(filenames)
}

// waitForInstallToComplete waits for the Verrazzano install to complete and shows the logs of
// the ongoing Verrazzano install.
func waitForInstallToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, vpoPodName string, namespacedName types.NamespacedName, timeout time.Duration, logFormat cmdhelpers.LogFormat) error {
	return cmdhelpers.WaitForOperationToComplete(client, kubeClient, vzHelper, vpoPodName, namespacedName, timeout, logFormat, vzapi.CondInstallComplete)
}
