// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"bufio"
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"time"

	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
# Uninstall Verrazzano except for CRDs and stream the logs to the console.  Stream the logs to the console until the uninstall completes.
vz uninstall

# Uninstall Verrazzano including the CRDs and wait for the command to complete. Output the logs in json format, timeout the command after 20 minutes.
vz uninstall --crds --timeout 20m`
)

func NewCmdUninstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUninstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)

	// Remove CRD's flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(crdsFlag, false, crdsFlagHelp)
	_ = cmd.PersistentFlags().MarkHidden(crdsFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an uninstall.")
	_ = cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	return cmd
}

func runCmdUninstall(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), "Not implemented yet\n")

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

	// Get the verrazzano install resource.
	vz, err = helpers.GetVerrazzanoResource(client, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name})
	if err != nil {
		return err
	}

	// Delete the Verrazzano custom resource.
	err = client.Delete(context.TODO(), vz)
	if err != nil {
		return fmt.Errorf("Failed to uninstall in Verrazzano resource: %s", err.Error())
	}

	uninstallPodName, err := waitForUninstallPod(client, vzHelper, vzapi.CondUpgradeComplete)
	if err != nil {
		return err
	}

	// Wait for the Verrazzano uninstall to complete.
	return waitForUninstallToComplete(client, kubeClient, vzHelper, uninstallPodName, types.NamespacedName{Namespace: vz.Namespace, Name: vz.Name}, timeout)
}

// WaitForPlatformOperator waits for the verrazzano-platform-operator to be ready
func WaitForUninstallPod(client client.Client, vzHelper helpers.VZHelper, condType vzapi.ConditionType) (string, error) {
	// Find the verrazzano-platform-operator using the app label selector
	appLabel, _ := labels.NewRequirement("app", selection.Equals, []string{constants.VerrazzanoPlatformOperator})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*appLabel)
	podList := corev1.PodList{}

	// Wait for the verrazzano-platform-operator pod to be found
	seconds := 0
	retryCount := 0
	for {
		retryCount++
		if retryCount > vpoWaitRetries {
			return "", fmt.Errorf("%s pod not found in namespace %s", constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
		}
		time.Sleep(constants.VerrazzanoPlatformOperatorWait * time.Second)
		seconds += constants.VerrazzanoPlatformOperatorWait

		err := client.List(
			context.TODO(),
			&podList,
			&client.ListOptions{
				Namespace:     vzconstants.VerrazzanoInstallNamespace,
				LabelSelector: labelSelector,
			})
		if err != nil {
			return "", fmt.Errorf("Failed to list pods %v", err)
		}
		if len(podList.Items) == 0 {
			continue
		}
		if len(podList.Items) > 1 {
			return "", fmt.Errorf("More than one %s pod was found in namespace %s", constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
		}
		break
	}

	// We found the verrazzano-platform-operator pod. Wait until it's containers are ready.
	pod := &corev1.Pod{}
	seconds = 0
	for {
		time.Sleep(constants.VerrazzanoPlatformOperatorWait * time.Second)
		seconds += constants.VerrazzanoPlatformOperatorWait
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("\rWaiting for verrazzano-platform-operator to be ready before starting %s - %d seconds", getOperationString(condType), seconds))

		err := client.Get(context.TODO(), types.NamespacedName{Namespace: podList.Items[0].Namespace, Name: podList.Items[0].Name}, pod)
		if err != nil {
			return "", err
		}
		initReady := true
		for _, initContainer := range pod.Status.InitContainerStatuses {
			if !initContainer.Ready {
				initReady = false
				break
			}
		}
		ready := true
		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready {
				ready = false
				break
			}
		}

		if initReady && ready {
			fmt.Fprintf(vzHelper.GetOutputStream(), "\n")
			break
		}
	}
	return pod.Name, nil
}

func waitForUninstallToComplete(client client.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, uninstallPodName string, namespacedName types.NamespacedName, timeout time.Duration) error {
	// Stream the logs from the uninstall job starting at the current time.
	sinceTime := metav1.Now()
	rc, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstallNamespace).GetLogs(uninstallPodName, &corev1.PodLogOptions{
		Container: "uninstall",
		Follow:    true,
		SinceTime: &sinceTime,
	}).Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("Failed to get logs stream: %v", err)
	}
	defer rc.Close()

	resChan := make(chan error, 1)
	go func() {
		sc := bufio.NewScanner(rc)
		sc.Split(bufio.ScanLines)
		for sc.Scan() {
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("%s\n", sc.Text()))

			// Return when the Verrazzano uninstall has completed
			vz, err := helpers.GetVerrazzanoResource(client, namespacedName)
			if err != nil {
				resChan <- err
			}
			for _, condition := range vz.Status.Conditions {
				// Operation condition met for uninstall
				if condition.Type == vzapi.CondUninstallComplete {
					resChan <- nil
				}
			}
		}
	}()

	select {
	case result := <-resChan:
		return result
	case <-time.After(timeout):
		if timeout.Nanoseconds() != 0 {
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Timeout %v exceeded waiting for uninstall to complete\n", timeout.String()))
		}
	}

	return nil
}
