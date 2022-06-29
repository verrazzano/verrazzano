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
	clik8sutil "github.com/verrazzano/verrazzano/tools/vz/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Number of retries after waiting a second for VPO to be ready
const vpoDefaultWaitRetries = 60 * 5

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
func WaitForPlatformOperator(client clipkg.Client, vzHelper helpers.VZHelper, condType vzapi.ConditionType, lastTransitionTime metav1.Time) (string, error) {
	// Find the verrazzano-platform-operator using the app label selector
	appLabel, _ := labels.NewRequirement("app", selection.Equals, []string{constants.VerrazzanoPlatformOperator})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*appLabel)
	podList := corev1.PodList{}

	deployments := []types.NamespacedName{
		{
			Name:      constants.VerrazzanoPlatformOperator,
			Namespace: vzconstants.VerrazzanoInstallNamespace,
		},
	}

	// Provide the user with feedback while waiting for the verrazzano-platform-operator to be ready
	feedbackChan := make(chan bool)
	defer close(feedbackChan)
	go func(outputStream io.Writer) {
		seconds := 0
		for {
			select {
			case <-feedbackChan:
				fmt.Fprint(outputStream, "\n")
				return
			default:
				time.Sleep(constants.VerrazzanoPlatformOperatorWait * time.Second)
				seconds += constants.VerrazzanoPlatformOperatorWait
				fmt.Fprintf(outputStream, fmt.Sprintf("\rWaiting for %s to be ready before starting %s - %d seconds", constants.VerrazzanoPlatformOperator, getOperationString(condType), seconds))
			}
		}
	}(vzHelper.GetOutputStream())

	// Wait for the verrazzano-platform-operator pod to be found
	seconds := 0
	retryCount := 0
	for {
		ready, err := clik8sutil.DeploymentsAreReady(client, deployments, 1, lastTransitionTime)
		if ready {
			break
		}

		retryCount++
		if retryCount > vpoWaitRetries {
			feedbackChan <- true
			return "", fmt.Errorf("Waiting for %s pod in namespace %s: %v", constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace, err)
		}
		time.Sleep(constants.VerrazzanoPlatformOperatorWait * time.Second)
		seconds += constants.VerrazzanoPlatformOperatorWait
	}
	feedbackChan <- true

	// Return the platform operator pod name
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
		return "", fmt.Errorf("Failed to find the Verrazzano platform operator in namespace %s", vzconstants.VerrazzanoInstallNamespace)
	}
	if len(podList.Items) > 1 {
		return "", fmt.Errorf("Waiting for %s, more than one %s pod was found in namespace %s", constants.VerrazzanoPlatformOperator, constants.VerrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	}

	return podList.Items[0].Name, nil
}

// WaitForOperationToComplete waits for the Verrazzano install/upgrade to complete and
// shows the logs of the ongoing Verrazzano install/upgrade.
func WaitForOperationToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, vpoPodName string, namespacedName types.NamespacedName, timeout time.Duration, logFormat LogFormat, condType vzapi.ConditionType) error {
	// Tail the log messages from the verrazzano-platform-operator starting at the current time.
	//
	// The stream is intentionally not closed due to not being able to cancel a blocking read.  The calls to
	// read input from this stream (sc.Scan) are blocking.  If you try to close the stream, it hangs until the
	// next read is satisfied, which may never occur if there is no more log output.
	sinceTime := metav1.Now()
	rc, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstallNamespace).GetLogs(vpoPodName, &corev1.PodLogOptions{
		Container: constants.VerrazzanoPlatformOperator,
		Follow:    true,
		SinceTime: &sinceTime,
	}).Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("Failed to read the %s log file: %s", constants.VerrazzanoPlatformOperator, err.Error())
	}

	resChan := make(chan error, 1)
	defer close(resChan)

	feedbackChan := make(chan bool)
	defer close(feedbackChan)

	// goroutine to stream log file output - this goroutine will be left running when this
	// function is exited because there is no way to cancel the blocking read to the input stream.
	re := regexp.MustCompile(`"level":"(.*?)","@timestamp":"(.*?)",(.*?)"message":"(.*?)",`)
	go func(outputStream io.Writer) {
		sc := bufio.NewScanner(rc)
		sc.Split(bufio.ScanLines)
		for {
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
	}(vzHelper.GetOutputStream())

	// goroutine to wait for the completion of the operation
	go func() {
		for {
			// Pause before each status check
			time.Sleep(1 * time.Second)
			select {
			case <-feedbackChan:
				return
			default:
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
			}
		}
	}()

	select {
	case result := <-resChan:
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
