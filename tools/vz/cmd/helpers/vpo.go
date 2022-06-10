// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"os"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const verrazzanoPlatformOperatorWait = 1

// Number of retries after waiting a second for VPO to be ready
var vpoWaitRetries = 60

// Used with unit testing
func SetVpoWaitRetries(retries int) { vpoWaitRetries = retries }
func ResetVpoWaitRetries()          { vpoWaitRetries = 60 }

// ApplyPlatformOperatorYaml applies a given version of the platform operator yaml file
func ApplyPlatformOperatorYaml(cmd *cobra.Command, client clipkg.Client, vzHelper helpers.VZHelper, version string) error {
	// Was an operator-file passed on the command line?
	operatorFile, err := GetOperatorFile(cmd)
	if err != nil {
		return fmt.Errorf("Failed to parse the command-line option %s: %s", constants.OperatorFileFlag, err.Error())
	}

	// If the operatorFile was specified, is it a local or remote file?
	url := ""
	internalFilename := ""
	if len(operatorFile) > 0 {
		if strings.HasPrefix(strings.ToLower(operatorFile), "https://") {
			url = operatorFile
		} else {
			internalFilename = operatorFile
		}
	} else {
		url = fmt.Sprintf(constants.VerrazzanoOperatorURL, version)
	}

	userVisibleFilename := operatorFile
	if len(url) > 0 {
		userVisibleFilename = url
		// Get the Verrazzano operator.yaml and store it in a temp file
		httpClient := vzHelper.GetHTTPClient()
		resp, err := httpClient.Get(url)
		if err != nil {
			return fmt.Errorf("Failed to access the Verrazzano operator.yaml file %s: %s", userVisibleFilename, err.Error())
		}
		// Store response in a temporary file
		tmpFile, err := ioutil.TempFile("", "vz")
		if err != nil {
			return fmt.Errorf("Failed to install the Verrazzano operator.yaml file %s: %s", userVisibleFilename, err.Error())
		}
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.ReadFrom(resp.Body)
		if err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("Failed to install the Verrazzano operator.yaml file %s: %s", userVisibleFilename, err.Error())
		}
		internalFilename = tmpFile.Name()
	}

	// Apply the Verrazzano operator.yaml
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Applying the file %s\n", userVisibleFilename))
	yamlApplier := k8sutil.NewYAMLApplier(client, "")
	err = yamlApplier.ApplyF(internalFilename)
	if err != nil {
		return fmt.Errorf("Failed to apply the file: %s", err.Error())
	}
	return nil
}

// WaitForPlatformOperator waits for the verrazzano-platform-operator to be ready
func WaitForPlatformOperator(client clipkg.Client, vzHelper helpers.VZHelper) (string, error) {
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
		time.Sleep(verrazzanoPlatformOperatorWait * time.Second)
		seconds += verrazzanoPlatformOperatorWait

		err := client.List(
			context.TODO(),
			&podList,
			&clipkg.ListOptions{
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
		time.Sleep(verrazzanoPlatformOperatorWait * time.Second)
		seconds += verrazzanoPlatformOperatorWait
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("\rWaiting for verrazzano-platform-operator to be ready before starting install - %d seconds", seconds))

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
