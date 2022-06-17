// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Number of retries after waiting a second for VPO to be ready
const vpoDefaultWaitRetries = 60

var vpoWaitRetries = vpoDefaultWaitRetries

// Used with unit testing
func SetVpoWaitRetries(retries int) { vpoWaitRetries = retries }
func ResetVpoWaitRetries()          { vpoWaitRetries = vpoDefaultWaitRetries }

// ApplyPlatformOperatorYaml applies a given version of the platform operator yaml file
func ApplyPlatformOperatorYaml(cmd *cobra.Command, client clipkg.Client, vzHelper helpers.VZHelper, version string) error {
	// Was an operator-file passed on the command line?
	operatorFile, err := GetOperatorFile(cmd)
	if err != nil {
		return err
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

	const accessErrorMsg = "Failed to access the Verrazzano operator.yaml file %s: %s"
	const applyErrorMsg = "Failed to apply the Verrazzano operator.yaml file %s: %s"
	userVisibleFilename := operatorFile
	if len(url) > 0 {
		userVisibleFilename = url
		// Get the Verrazzano operator.yaml and store it in a temp file
		httpClient := vzHelper.GetHTTPClient()
		resp, err := httpClient.Get(url)
		if err != nil {
			return fmt.Errorf(accessErrorMsg, userVisibleFilename, err.Error())
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf(accessErrorMsg, userVisibleFilename, resp.Status)
		}
		// Store response in a temporary file
		tmpFile, err := ioutil.TempFile("", "vz")
		if err != nil {
			return fmt.Errorf(applyErrorMsg, userVisibleFilename, err.Error())
		}
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.ReadFrom(resp.Body)
		if err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf(applyErrorMsg, userVisibleFilename, err.Error())
		}
		internalFilename = tmpFile.Name()
	}

	// Apply the Verrazzano operator.yaml
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Applying the file %s\n", userVisibleFilename))
	yamlApplier := k8sutil.NewYAMLApplier(client, "")
	err = yamlApplier.ApplyF(internalFilename)
	if err != nil {
		return fmt.Errorf(applyErrorMsg, internalFilename, err.Error())
	}

	// Dump out the object result messages
	for _, result := range yamlApplier.ObjectResultMsgs() {
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("%s\n", strings.ToLower(result)))
	}
	return nil
}

// WaitForPlatformOperator waits for the verrazzano-platform-operator to be ready
func WaitForPlatformOperator(client clipkg.Client, vzHelper helpers.VZHelper, condType vzapi.ConditionType) (string, error) {
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
			return "", fmt.Errorf("Waiting for %s, pod was not found in namespace %s", constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
		}
		time.Sleep(constants.VerrazzanoPlatformOperatorWait * time.Second)
		seconds += constants.VerrazzanoPlatformOperatorWait

		err := client.List(
			context.TODO(),
			&podList,
			&clipkg.ListOptions{
				Namespace:     vzconstants.VerrazzanoInstallNamespace,
				LabelSelector: labelSelector,
			})
		if err != nil {
			return "", fmt.Errorf("Waiting for %s, failed to list pods: %s", constants.VerrazzanoPlatformOperator, err.Error())
		}
		if len(podList.Items) == 0 {
			continue
		}
		if len(podList.Items) > 1 {
			return "", fmt.Errorf("Waiting for %s, more than one %s pod was found in namespace %s", constants.VerrazzanoPlatformOperator, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
		}
		break
	}

	// We found the verrazzano-platform-operator pod. Wait until it's containers are ready.
	pod := &corev1.Pod{}
	seconds = 0
	for {
		time.Sleep(constants.VerrazzanoPlatformOperatorWait * time.Second)
		seconds += constants.VerrazzanoPlatformOperatorWait
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("\rWaiting for %s to be ready before starting %s - %d seconds", constants.VerrazzanoPlatformOperator, getOperationString(condType), seconds))

		err := client.Get(context.TODO(), types.NamespacedName{Namespace: podList.Items[0].Namespace, Name: podList.Items[0].Name}, pod)
		if err != nil {
			return "", fmt.Errorf("Waiting for %s to be ready: %s", constants.VerrazzanoPlatformOperator, err.Error())
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

// WaitForOperationToComplete waits for the Verrazzano install/upgrade to complete and
// shows the logs of the ongoing Verrazzano install/upgrade.
func WaitForOperationToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, vpoPodName string, namespacedName types.NamespacedName, timeout time.Duration, logFormat LogFormat, condType vzapi.ConditionType) error {
	// Tail the log messages from the verrazzano-platform-operator starting at the current time.
	sinceTime := metav1.Now()
	rc, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstallNamespace).GetLogs(vpoPodName, &corev1.PodLogOptions{
		Container: constants.VerrazzanoPlatformOperator,
		Follow:    true,
		SinceTime: &sinceTime,
	}).Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("Failed to read the %s log file: %s", constants.VerrazzanoPlatformOperator, err.Error())
	}
	defer rc.Close()

	// Create the channels
	logChanQuit := make(chan bool)
	defer close(logChanQuit)

	resChan := make(chan error, 1)
	defer close(resChan)

	// goroutine to stream log file output
	re := regexp.MustCompile(`"level":"(.*?)","@timestamp":"(.*?)",(.*?)"message":"(.*?)",`)
	go func(outputStream io.Writer) {
		sc := bufio.NewScanner(rc)
		sc.Split(bufio.ScanLines)
		for {
			select {
			case <-logChanQuit:
				return
			default:
				sc.Scan()
				if logFormat == LogFormatSimple {
					res := re.FindAllStringSubmatch(sc.Text(), -1)
					// res[0][2] is the timestamp
					// res[0][1] is the level
					// res[0][4] is the message
					if res != nil {
						// Print each log message in the form "timestamp level message".
						// For example, "2022-06-03T00:05:10.042Z info Component keycloak successfully installed"
						fmt.Fprintf(outputStream, fmt.Sprintf("%s %s %s\n", res[0][2], res[0][1], res[0][4]))
					}
				} else if logFormat == LogFormatJSON {
					fmt.Fprintf(outputStream, fmt.Sprintf("%s\n", sc.Text()))
				}
			}
		}
	}(vzHelper.GetOutputStream())

	// goroutine to wait for the completion of the operation
	go func() {
		for {
			// Return when the Verrazzano operation has completed
			vz, err := helpers.GetVerrazzanoResource(client, namespacedName)
			if err != nil {
				resChan <- err
				return
			}
			for _, condition := range vz.Status.Conditions {
				// Operation condition met for install/upgrade
				if condition.Type == condType {
					resChan <- nil
					return
				}
			}

			// Pause before next status check
			time.Sleep(15 * time.Second)
		}
	}()

	select {
	case result := <-resChan:
		logChanQuit <- true
		return result
	case <-time.After(timeout):
		if timeout.Nanoseconds() != 0 {
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Timeout %v exceeded waiting for %s to complete\n", timeout.String(), getOperationString(condType)))
		}
	}

	return nil
}

// return the operation string to display
func getOperationString(condType vzapi.ConditionType) string {
	operation := "install"
	if condType == vzapi.CondUpgradeComplete {
		operation = "upgrade"
	}
	return operation
}
