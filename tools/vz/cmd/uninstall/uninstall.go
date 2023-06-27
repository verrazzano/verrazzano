// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/bugreport"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	adminv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	ConfirmUninstallFlag          = "skip-confirmation"
	ConfirmUninstallFlagShorthand = "y"
)

// Number of retries after waiting a second for uninstall job pod to be ready
const uninstallDefaultWaitRetries = 300
const verrazzanoUninstallJobDetectWait = 1

var uninstallWaitRetries = uninstallDefaultWaitRetries

// Used with unit testing
func setWaitRetries(retries int) { uninstallWaitRetries = retries }
func resetWaitRetries()          { uninstallWaitRetries = uninstallDefaultWaitRetries }

var propagationPolicy = metav1.DeletePropagationBackground
var deleteOptions = &client.DeleteOptions{PropagationPolicy: &propagationPolicy}

var logsEnum = cmdhelpers.LogFormatSimple

func NewCmdUninstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUninstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().Duration(constants.VPOTimeoutFlag, time.Minute*5, constants.VPOTimeoutFlagHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)
	cmd.PersistentFlags().Bool(constants.AutoBugReportFlag, constants.AutoBugReportFlagDefault, constants.AutoBugReportFlagHelp)

	// Remove CRD's flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(crdsFlag, false, crdsFlagHelp)
	_ = cmd.PersistentFlags().MarkHidden(crdsFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an uninstall.")
	_ = cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	// Hide the flag for overriding the default wait timeout for the platform-operator
	cmd.PersistentFlags().MarkHidden(constants.VPOTimeoutFlag)

	// When set to false, uninstall prompt can be suppressed
	cmd.PersistentFlags().BoolP(constants.SkipConfirmationFlag, constants.SkipConfirmationShort, false, "Used to confirm uninstall and suppress prompt")
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

	confirmUninstallFlag, err := cmd.Flags().GetBool(ConfirmUninstallFlag)
	continueUninstall, err := continueUninstall(confirmUninstallFlag)
	if err != nil {
		return err
	}
	if !continueUninstall {
		return nil
	}

	// Decide whether to stream the old uninstall job log or the VPO log.  With Verrazzano 1.4.0,
	// the uninstall job has been removed and the VPO does the uninstall.
	useUninstallJob, err := cmdhelpers.UsePlatformOperatorUninstallJob(client)
	if err != nil {
		return err
	}
	if useUninstallJob {
		// log-format argument ignored with pre 1.4.0 uninstalls if specified
		if cmd.PersistentFlags().Changed(constants.LogFormatFlag) {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Warning: --log-format argument is ignored with uninstalls prior to v1.4.0\n")
		}
	}

	// Get the kubernetes clientset.  This will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Get the timeout value for the uninstall command.
	timeout, err := cmdhelpers.GetWaitTimeout(cmd, constants.TimeoutFlag)
	if err != nil {
		return err
	}

	// Get the VPO timeout
	vpoTimeout, err := cmdhelpers.GetWaitTimeout(cmd, constants.VPOTimeoutFlag)
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
		// Try to delete the resource as v1alpha1 if the v1beta1 API version did not match
		if meta.IsNoMatchError(err) {
			vzV1Alpha1 := &v1alpha1.Verrazzano{}
			err = vzV1Alpha1.ConvertFrom(vz)
			if err != nil {
				return failedToUninstallErr(err)
			}
			if err := client.Delete(context.TODO(), vzV1Alpha1); err != nil {
				return failedToUninstallErr(err)
			}
		} else {
			return bugreport.AutoBugReport(cmd, vzHelper, err)
		}
	}
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "Uninstalling Verrazzano\n")

	// Wait for the Verrazzano uninstall to complete.
	err = waitForUninstallToComplete(client, kubeClient, vzHelper, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, timeout, vpoTimeout, logFormat, useUninstallJob)
	if err != nil {
		return bugreport.AutoBugReport(cmd, vzHelper, err)
	}
	return nil
}

// cleanupResources deletes remaining resources that remain after the Verrazzano resource in uninstalled
// Resources that fail to delete will log an error but will not return
func cleanupResources(client client.Client, vzHelper helpers.VZHelper) {
	// Delete verrazzano-install namespace
	err := deleteNamespace(client, constants.VerrazzanoInstall)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	// Delete other verrazzano resources
	err = deleteWebhookConfiguration(client, constants.VerrazzanoPlatformOperatorWebhook)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	err = deleteWebhookConfiguration(client, constants.VerrazzanoMysqlInstallValuesWebhook)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	err = deleteWebhookConfiguration(client, constants.VerrazzanoRequirementsValidatorWebhook)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}

	err = deleteMutatingWebhookConfiguration(client, constants.MysqlBackupMutatingWebhookName)
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

	err = deleteClusterRole(client, vzconstants.VerrazzanoClusterRancherName)
	if err != nil {
		_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), err.Error()+"\n")
	}
}

// getUninstallJobPodName returns the name of the pod for the verrazzano-uninstall job
// The uninstall job is triggered by deleting the Verrazzano custom resource
func getUninstallJobPodName(c client.Client, vzHelper helpers.VZHelper, jobName string) (string, error) {
	// Find the verrazzano-uninstall pod using the job-name label selector
	jobNameLabel, _ := labels.NewRequirement("job-name", selection.Equals, []string{jobName})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*jobNameLabel)
	podList := corev1.PodList{}

	// Provide the user with feedback while waiting for the verrazzano-uninstall pod to be ready
	feedbackChan := make(chan bool)
	defer close(feedbackChan)
	go func(outputStream io.Writer) {
		seconds := 0
		for {
			select {
			case <-feedbackChan:
				return
			default:
				time.Sleep(verrazzanoUninstallJobDetectWait * time.Second)
				seconds += verrazzanoUninstallJobDetectWait
				fmt.Fprintf(outputStream, fmt.Sprintf("\rWaiting for %s pod to be ready before starting uninstall - %d seconds", jobName, seconds))
			}
		}
	}(vzHelper.GetOutputStream())

	// Wait for the verrazzano-uninstall pod to be found
	seconds := 0
	retryCount := 0
	for {
		retryCount++
		if retryCount > uninstallWaitRetries {
			return "", fmt.Errorf("Waiting for %s, %s pod not found in namespace %s", jobName, jobName, vzconstants.VerrazzanoInstallNamespace)
		}
		time.Sleep(verrazzanoUninstallJobDetectWait * time.Second)
		seconds += verrazzanoUninstallJobDetectWait

		err := c.List(
			context.TODO(),
			&podList,
			&client.ListOptions{
				Namespace:     vzconstants.VerrazzanoInstallNamespace,
				LabelSelector: labelSelector,
			})
		if err != nil {
			return "", fmt.Errorf("Waiting for %s, failed to list pods: %s", jobName, err.Error())
		}
		if len(podList.Items) == 0 {
			continue
		}
		if len(podList.Items) > 1 {
			return "", fmt.Errorf("Waiting for %s, more than one %s pod was found in namespace %s", jobName, jobName, vzconstants.VerrazzanoInstallNamespace)
		}
		feedbackChan <- true
		break
	}

	// We found the verrazzano-uninstall pod. Wait until it's containers are ready.
	pod := &corev1.Pod{}
	seconds = 0
	for {
		time.Sleep(verrazzanoUninstallJobDetectWait * time.Second)
		seconds += verrazzanoUninstallJobDetectWait

		err := c.Get(context.TODO(), types.NamespacedName{Namespace: podList.Items[0].Namespace, Name: podList.Items[0].Name}, pod)
		if err != nil {
			return "", err
		}

		ready := true
		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready {
				ready = false
				break
			}
		}

		if ready {
			_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "\n")
			break
		}
	}
	return pod.Name, nil
}

// waitForUninstallToComplete waits for the Verrazzano resource to no longer exist
func waitForUninstallToComplete(client client.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, namespacedName types.NamespacedName, timeout time.Duration, vpoTimeout time.Duration, logFormat cmdhelpers.LogFormat, useUninstallJob bool) error {
	resChan := make(chan error, 1)
	defer close(resChan)

	feedbackChan := make(chan bool)
	defer close(feedbackChan)

	rc, err := getScanner(useUninstallJob, client, kubeClient, vzHelper, namespacedName)
	if err != nil {
		return err
	}

	go func(outputStream io.Writer, sc *bufio.Scanner, useUninstallJob bool) {
		re := regexp.MustCompile(cmdhelpers.VpoSimpleLogFormatRegexp)
		var err error
		secondsWaited := 0
		maxSecondsToWait := int(vpoTimeout.Seconds())
		const secondsPerRetry = 10

		for {
			if sc == nil {
				sc, err = getScanner(useUninstallJob, client, kubeClient, vzHelper, namespacedName)
				if err != nil {
					fmt.Fprintf(outputStream, fmt.Sprintf("Failed to connect to the uninstall output, waited %d of %d seconds to recover: %v\n", secondsWaited, maxSecondsToWait, err))
					secondsWaited += secondsPerRetry
					if secondsWaited > maxSecondsToWait {
						return
					}
					time.Sleep(secondsPerRetry * time.Second)
					continue
				}
				secondsWaited = 0
				sc.Split(bufio.ScanLines)
			}

			scannedOk := sc.Scan()
			if !scannedOk {
				errText := ""
				if sc.Err() != nil {
					errText = fmt.Sprintf(": %v", sc.Err())
				}
				fmt.Fprintf(outputStream, fmt.Sprintf("Lost connection to the uninstall output, attempting to reconnect%s\n", errText))
				sc = nil
				continue
			}

			if !useUninstallJob && logFormat == cmdhelpers.LogFormatSimple {
				cmdhelpers.PrintSimpleLogFormat(sc, outputStream, re)
			} else {
				_, _ = fmt.Fprintf(outputStream, fmt.Sprintf("%s\n", sc.Text()))
			}
		}
	}(vzHelper.GetOutputStream(), rc, useUninstallJob)

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

	var timeoutErr error
	select {
	case result := <-resChan:
		if result == nil {
			// Delete remaining Verrazzano resources, excluding CRDs
			cleanupResources(client, vzHelper)
		}
		return result
	case <-time.After(timeout):
		if timeout.Nanoseconds() != 0 {
			feedbackChan <- true
			timeoutErr = fmt.Errorf("Timeout %v exceeded waiting for uninstall to complete", timeout.String())
		}
	}
	return timeoutErr
}

// getScanner - get scanner for uninstall console output
func getScanner(useUninstallJob bool, client client.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, namespacedName types.NamespacedName) (*bufio.Scanner, error) {
	var podName string
	var err error
	if useUninstallJob {
		// Get the uninstall job for streaming the logs
		jobName := constants.VerrazzanoUninstall + "-" + namespacedName.Name
		podName, err = getUninstallJobPodName(client, vzHelper, jobName)
	} else {
		// Get the VPO pod for streaming the logs
		podName, err = cmdhelpers.GetVerrazzanoPlatformOperatorPodName(client)
	}
	if err != nil {
		return nil, err
	}

	var rc io.ReadCloser
	if useUninstallJob {
		rc, err = getUninstallJobLogStream(kubeClient, podName)
	} else {
		rc, err = cmdhelpers.GetVpoLogStream(kubeClient, podName)
	}
	if err != nil {
		return nil, err
	}

	return bufio.NewScanner(rc), nil
}

// getUninstallJobLogStream returns the stream to the uninstall job log file
func getUninstallJobLogStream(kubeClient kubernetes.Interface, uninstallPodName string) (io.ReadCloser, error) {
	// Tail the log messages from the uninstall job log starting at the current time.
	sinceTime := metav1.Now()
	rc, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstallNamespace).GetLogs(uninstallPodName, &corev1.PodLogOptions{
		Container: "uninstall",
		Follow:    true,
		SinceTime: &sinceTime,
	}).Stream(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("Failed to read the %s log file: %s", uninstallPodName, err.Error())
	}
	return rc, nil
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
func deleteWebhookConfiguration(client client.Client, name string) error {
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

// deleteMutatingWebhookConfiguration deletes a given MutatingWebhookConfiguration
func deleteMutatingWebhookConfiguration(client client.Client, name string) error {
	mwc := &adminv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := client.Delete(context.TODO(), mwc, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("Failed to delete MutatingWebhookConfiguration resource %s: %s", name, err.Error())
	}
	return nil
}

// deleteClusterRoleBinding deletes a given ClusterRoleBinding
func deleteClusterRoleBinding(client client.Client, name string) error {
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

func failedToUninstallErr(err error) error {
	return fmt.Errorf("Failed to uninstall Verrazzano: %s", err.Error())
}

func continueUninstall(confirmUninstall bool) (bool, error) {
	if confirmUninstall {
		return true, nil
	}
	var response string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Are you sure you want to uninstall Verrazzano? [y/N]: ")
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
