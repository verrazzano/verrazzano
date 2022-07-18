// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	adminv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CommandName  = "uninstall"
	crdsFlag     = "crds"
	crdsFlagHelp = "Completely remove all CRDs that were installed by Verrazzano"
	helpShort    = "Uninstall Verrazzano"
	helpLong     = `Uninstall the Verrazzano Platform Operator and all of the currently installed components`
	helpExample  = `
# Uninstall Verrazzano and stream the logs to the console.  Stream the logs to the console until the uninstall completes.
vz uninstall

# Uninstall Verrazzano and wait for the command to complete. Timeout the command after 30 minutes.
vz uninstall --timeout 30m`
)

var propagationPolicy = metav1.DeletePropagationBackground
var deleteOptions = &clipkg.DeleteOptions{PropagationPolicy: &propagationPolicy}

var logsEnum = cmdhelpers.LogFormatSimple

func NewCmdUninstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUninstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)

	// Remove CRD's flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(crdsFlag, false, crdsFlagHelp)
	_ = cmd.PersistentFlags().MarkHidden(crdsFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an uninstall.")
	_ = cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	return cmd
}

func runCmdUninstall(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {

	// Get the controller runtime client.
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Find the Verrazzano resource to uninstall.
	vz, err := helpers.FindVerrazzanoResource(client)
	if err != nil {
		return fmt.Errorf("Verrazzano is not installed: %s", err.Error())
	}

	// Get the kubernetes clientset.  This will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Get the timeout value for the uninstall command.
	timeout, err := cmdhelpers.GetWaitTimeout(cmd)
	if err != nil {
		return err
	}

	// Get the log format value
	logFormat, err := cmdhelpers.GetLogFormat(cmd)
	if err != nil {
		return err
	}

	// Delete the Verrazzano custom resource.
	err = client.Delete(context.TODO(), vz)
	if err != nil {
		return fmt.Errorf("Failed to uninstall Verrazzano: %s", err.Error())
	}
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "Uninstalling Verrazzano\n")

	// Get the VPO pod to stream the logs from.
	vpoPodName, err := cmdhelpers.GetVerrazzanoPlatformOperatorPodName(client)
	if err != nil {
		return err
	}

	// Wait for the Verrazzano uninstall to complete.
	err = waitForUninstallToComplete(client, kubeClient, vzHelper, vpoPodName, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, timeout, logFormat)
	if err != nil {
		return fmt.Errorf("Failed to uninstall Verrazzano: %s", err.Error())
	}

	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "Successfully uninstalled Verrazzano\n")
	return nil
}

// cleanupResources deletes remaining resources that remain after the Verrazzano resource in uninstalled
// Resources that fail to delete will log an error but will not return
func cleanupResources(client clipkg.Client, vzHelper helpers.VZHelper) error {
	// Delete verrazzano-install namespace
	err := deleteNamespace(client, constants.VerrazzanoInstall)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	// Delete other verrazzano resources
	err = deleteWebhookConfiguration(client, constants.VerrazzanoPlatformOperator)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	err = deleteClusterRoleBinding(client, constants.VerrazzanoPlatformOperator)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	err = deleteClusterRole(client, constants.VerrazzanoManagedCluster)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}
	return nil
}

// waitForUninstallToComplete waits for the Verrazzano resource to no longer exist
func waitForUninstallToComplete(client client.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, vpoPodName string, namespacedName types.NamespacedName, timeout time.Duration, logFormat cmdhelpers.LogFormat) error {
	rc, err := cmdhelpers.GetLogStream(kubeClient, vpoPodName)
	if err != nil {
		return err
	}

	resChan := make(chan error, 1)
	defer close(resChan)

	feedbackChan := make(chan bool)
	defer close(feedbackChan)

	re := regexp.MustCompile(cmdhelpers.VpoSimpleLogFormatRegexp)
	go func(outputStream io.Writer) {
		sc := bufio.NewScanner(rc)
		sc.Split(bufio.ScanLines)
		for sc.Scan() {
			if logFormat == cmdhelpers.LogFormatSimple {
				cmdhelpers.PrintSimpleLogFormat(sc, outputStream, re)
			} else if logFormat == cmdhelpers.LogFormatJSON {
				_, _ = fmt.Fprintf(outputStream, fmt.Sprintf("%s\n", sc.Text()))
			}
		}
	}(vzHelper.GetOutputStream())

	go func() {
		for {
			// Pause before each check
			time.Sleep(1 * time.Second)
			select {
			case <-feedbackChan:
				return
			default:
				// Return when the Verrazzano uninstall has completed
				vz, err := helpers.GetVerrazzanoResource(client, namespacedName)
				if vz == nil {
					resChan <- nil
					return
				}
				if err != nil && !errors.IsNotFound(err) {
					resChan <- err
					return
				}
			}
		}
	}()

	select {
	case result := <-resChan:
		// Delete remaining Verrazzano resources, excluding CRDs
		_ = cleanupResources(client, vzHelper)
		return result
	case <-time.After(timeout):
		if timeout.Nanoseconds() != 0 {
			feedbackChan <- true
			_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Timeout %v exceeded waiting for uninstall to complete\n", timeout.String()))
		}
	}
	// Delete remaining Verrazzano resources, excluding CRDs
	_ = cleanupResources(client, vzHelper)
	return nil
}

// deleteNamespace deletes a given Namespace
func deleteNamespace(client client.Client, name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := client.Delete(context.TODO(), ns, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to delete Namespace resource %s: %s", name, err.Error())
	}
	return nil
}

// deleteWebhookConfiguration deletes a given ValidatingWebhookConfiguration
func deleteWebhookConfiguration(client clipkg.Client, name string) error {
	vwc := &adminv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := client.Delete(context.TODO(), vwc, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to delete ValidatingWebhookConfiguration resource %s: %s", name, err.Error())
	}
	return nil
}

// deleteClusterRoleBinding deletes a given ClusterRoleBinding
func deleteClusterRoleBinding(client clipkg.Client, name string) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := client.Delete(context.TODO(), crb, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to delete ClusterRoleBinding resource %s: %s", name, err.Error())
	}
	return nil
}

// deleteClusterRole deletes a given ClusterRole
func deleteClusterRole(client client.Client, name string) error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := client.Delete(context.TODO(), cr, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to delete ClusterRole resource %s: %s", name, err.Error())
	}
	return nil
}
