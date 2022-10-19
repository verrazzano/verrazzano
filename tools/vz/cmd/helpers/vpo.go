// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const VpoSimpleLogFormatRegexp = `"level":"(.*?)","@timestamp":"(.*?)",(.*?)"message":"(.*?)",`

// Number of retries after waiting a second for VPO to be ready
const vpoDefaultWaitRetries = 60 * 50

var vpoWaitRetries = vpoDefaultWaitRetries

// Used with unit testing
func SetVpoWaitRetries(retries int) { vpoWaitRetries = retries }
func ResetVpoWaitRetries()          { vpoWaitRetries = vpoDefaultWaitRetries }

// deleteLeftoverPlatformOperatorSig is a function needed for unit test override
type deleteLeftoverPlatformOperatorSig func(client clipkg.Client) error

// DeleteFunc is the default deleteLeftoverPlatformOperator function
var DeleteFunc deleteLeftoverPlatformOperatorSig = deleteLeftoverPlatformOperator

func SetDeleteFunc(f deleteLeftoverPlatformOperatorSig) {
	DeleteFunc = f
}

func SetDefaultDeleteFunc() {
	DeleteFunc = deleteLeftoverPlatformOperator
}

func FakeDeleteFunc(client clipkg.Client) error {
	return nil
}

// UsePlatformOperatorUninstallJob determines whether the version of the platform operator is using an uninstall job.
// The uninstall job was removed with Verrazzano 1.4.0.
func UsePlatformOperatorUninstallJob(client clipkg.Client) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := client.Get(context.TODO(), types.NamespacedName{Namespace: vzconstants.VerrazzanoInstallNamespace, Name: constants.VerrazzanoPlatformOperator}, deployment)
	if err != nil {
		return false, fmt.Errorf("Failed to find %s/%s: %s", vzconstants.VerrazzanoInstallNamespace, constants.VerrazzanoPlatformOperator, err.Error())
	}

	// label does not exist therefore uninstall job is being used
	version, ok := deployment.Labels["app.kubernetes.io/version"]
	if !ok {
		return true, nil
	}

	minVersion := semver.SemVersion{Major: 1, Minor: 4, Patch: 0}
	vzVersion, err := semver.NewSemVersion(version)
	if err != nil {
		return false, err
	}

	// Version of platform operator is less than  1.4.0 therefore uninstall job is being used
	if vzVersion.IsLessThan(&minVersion) {
		return true, nil
	}

	return false, nil
}

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
		url, err = helpers.GetOperatorYaml(version)
		if err != nil {
			return err
		}
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
		tmpFile, err := os.CreateTemp("", "vz")
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
func WaitForPlatformOperator(client clipkg.Client, vzHelper helpers.VZHelper, condType v1beta1.ConditionType) (string, error) {
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
		ready, err := vpoIsReady(client)
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
	return GetVerrazzanoPlatformOperatorPodName(client)
}

// WaitForOperationToComplete waits for the Verrazzano install/upgrade to complete and
// shows the logs of the ongoing Verrazzano install/upgrade.
func WaitForOperationToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, vpoPodName string, namespacedName types.NamespacedName, timeout time.Duration, logFormat LogFormat, condType v1beta1.ConditionType) error {
	rc, err := GetVpoLogStream(kubeClient, vpoPodName)
	if err != nil {
		return err
	}
	resChan := make(chan error, 1)
	defer close(resChan)

	feedbackChan := make(chan bool)
	defer close(feedbackChan)

	// goroutine to stream log file output - this goroutine will be left running when this
	// function is exited because there is no way to cancel the blocking read to the input stream.
	re := regexp.MustCompile(VpoSimpleLogFormatRegexp)
	go func(outputStream io.Writer) {
		sc := bufio.NewScanner(rc)
		sc.Split(bufio.ScanLines)
		for {
			sc.Scan()
			if logFormat == LogFormatSimple {
				PrintSimpleLogFormat(sc, outputStream, re)
			} else if logFormat == LogFormatJSON {
				fmt.Fprintf(outputStream, fmt.Sprintf("%s\n", sc.Text()))
			}
		}
	}(vzHelper.GetOutputStream())

	startTime := time.Now().UTC()

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
					// Retry if there is a problem getting the resource.  It is ok to keep retrying since
					// WaitForOperationToComplete main routine will timeout.
					time.Sleep(10 * time.Second)
					continue
				}
				for _, condition := range vz.Status.Conditions {
					// Operation condition met for install/upgrade
					if condition.Type == condType {
						condTime, err := time.Parse(time.RFC3339, condition.LastTransitionTime)
						if err != nil {
							resChan <- fmt.Errorf("Failed parsing status condition lastTransitionTime: %s", err.Error())
							return
						}
						// There can be multiple conditions with the same type.  Make sure we find a match
						// beyond the start time.
						if condTime.After(startTime) {
							resChan <- nil
							return
						}
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
			return fmt.Errorf("Timeout %v exceeded waiting for %s to complete", timeout.String(), getOperationString(condType))
		}
	}

	return nil
}

// GetVerrazzanoPlatformOperatorPodName returns the VPO pod name
func GetVerrazzanoPlatformOperatorPodName(client clipkg.Client) (string, error) {
	appLabel, _ := labels.NewRequirement("app", selection.Equals, []string{constants.VerrazzanoPlatformOperator})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*appLabel)
	podList := corev1.PodList{}
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

// GetVpoLogStream returns the stream to the verrazzano-platform-operator log file
func GetVpoLogStream(kubeClient kubernetes.Interface, vpoPodName string) (io.ReadCloser, error) {
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
		return nil, fmt.Errorf("Failed to read the %s log file: %s", constants.VerrazzanoPlatformOperator, err.Error())
	}
	return rc, nil
}

// PrintSimpleLogFormat display a VPO log message with the simple log format
func PrintSimpleLogFormat(sc *bufio.Scanner, outputStream io.Writer, regexp *regexp.Regexp) {
	res := regexp.FindAllStringSubmatch(sc.Text(), -1)
	// res[0][2] is the timestamp
	// res[0][1] is the level
	// res[0][4] is the message
	if res != nil {
		// Print each log message in the form "timestamp level message".
		// For example, "2022-06-03T00:05:10.042Z info Component keycloak successfully installed"
		fmt.Fprintf(outputStream, fmt.Sprintf("%s %s %s\n", res[0][2], res[0][1], res[0][4]))
	}
}

// return the operation string to display
func getOperationString(condType v1beta1.ConditionType) string {
	operation := "install"
	if condType == v1beta1.CondUpgradeComplete {
		operation = "upgrade"
	}
	return operation
}

// vpoIsReady check that the named deployments have the minimum number of specified replicas ready and available
func vpoIsReady(client clipkg.Client) (bool, error) {
	var expectedReplicas int32 = 1
	deployment := appsv1.Deployment{}
	namespacedName := types.NamespacedName{Name: constants.VerrazzanoPlatformOperator, Namespace: vzconstants.VerrazzanoInstallNamespace}
	if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("Failed getting deployment %s: %s", constants.VerrazzanoPlatformOperator, err.Error())
	}
	if deployment.Status.UpdatedReplicas < expectedReplicas {
		return false, nil
	}
	if deployment.Status.AvailableReplicas < expectedReplicas {
		return false, nil
	}

	if !ready.PodsReadyDeployment(nil, client, namespacedName, deployment.Spec.Selector, expectedReplicas, constants.VerrazzanoPlatformOperator) {
		return false, nil
	}

	return true, nil
}

// deleteLeftoverPlatformOperator deletes leftover verrazzano-operator deployment after an abort.
// This allows for the verrazzano-operator validatingWebhookConfiguration to be updated with an updated caBundle.
func deleteLeftoverPlatformOperator(client clipkg.Client) error {
	vpoDeployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: vzconstants.VerrazzanoInstallNamespace,
			Name:      constants.VerrazzanoPlatformOperator,
		},
	}
	if err := client.Delete(context.TODO(), &vpoDeployment); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("Failed to delete leftover verrazzano-operator deployement: %s", err.Error())
		}
	}
	return nil
}
