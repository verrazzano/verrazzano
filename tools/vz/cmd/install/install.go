// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	verrazzanoPlatformOperator     = "verrazzano-platform-operator"
	verrazzanoPlatformOperatorWait = 1
)

var vpoWaitRetries = 60
var logsEnum = cmdhelpers.LogFormatSimple

// Use with unit testing
func resetVpoWaitRetries() { vpoWaitRetries = 60 }

func NewCmdInstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdInstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, constants.VersionFlagHelp)
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
	err := validateCmd(cmd)
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
		version, err = cmd.PersistentFlags().GetString(constants.VersionFlag)
		if err != nil {
			return err
		}
		if version == constants.VersionFlagDefault {
			// Find the latest release version of Verrazzano
			version, err = helpers.GetLatestReleaseVersion(vzHelper.GetHTTPClient())
			if err != nil {
				return err
			}
		}
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Installing Verrazzano version %s\n", version))
	}

	// Apply the Verrazzano operator.yaml.
	err = applyPlatformOperatorYaml(cmd, client, vzHelper, version)
	if err != nil {
		return err
	}

	// Wait for the platform operator to be ready before we create the Verrazzano resource.
	vpoPodName, err := waitForPlatformOperator(client, vzHelper)
	if err != nil {
		return err
	}

	// Create the Verrazzano install resource.
	err = client.Create(context.TODO(), vz)
	if err != nil {
		return err
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

// applyPlatformOperatorYaml applies a given version of the platform operator yaml file
func applyPlatformOperatorYaml(cmd *cobra.Command, client client.Client, vzHelper helpers.VZHelper, version string) error {
	// Was an operator-file passed on the command line?
	operatorFile, err := cmdhelpers.GetOperatorFile(cmd)
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

	// Dump out the object result messages
	for _, result := range yamlApplier.ObjectResultMsgs() {
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("%s\n", strings.ToLower(result)))
	}
	return nil
}

// waitForPlatformOperator waits for the verrazzano-platform-operator to be ready
func waitForPlatformOperator(client clipkg.Client, vzHelper helpers.VZHelper) (string, error) {
	// Find the verrazzano-platform-operator using the app label selector
	appLabel, _ := labels.NewRequirement("app", selection.Equals, []string{verrazzanoPlatformOperator})
	labelSelector := labels.NewSelector()
	labelSelector = labelSelector.Add(*appLabel)
	podList := corev1.PodList{}

	// Wait for the verrazzano-platform-operator pod to be found
	seconds := 0
	retryCount := 0
	for {
		retryCount++
		if retryCount > vpoWaitRetries {
			return "", fmt.Errorf("%s pod not found in namespace %s", verrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
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
			return "", fmt.Errorf("More than one %s pod was found in namespace %s", verrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
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

// waitForInstallToComplete waits for the Verrazzano install to complete and shows the logs of
// the ongoing Verrazzano install.
func waitForInstallToComplete(client clipkg.Client, kubeClient kubernetes.Interface, vzHelper helpers.VZHelper, vpoPodName string, namespacedName types.NamespacedName, timeout time.Duration, logFormat cmdhelpers.LogFormat) error {
	// Tail the log messages from the verrazzano-platform-operator starting at the current time.
	sinceTime := metav1.Now()
	rc, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstallNamespace).GetLogs(vpoPodName, &corev1.PodLogOptions{
		Container: verrazzanoPlatformOperator,
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
			if logFormat == cmdhelpers.LogFormatSimple {
				re := regexp.MustCompile(`"level":"(.*?)","@timestamp":"(.*?)",(.*?)"message":"(.*?)",`)
				res := re.FindAllStringSubmatch(sc.Text(), -1)
				// res[0][2] is the timestamp
				// res[0][1] is the level
				// res[0][4] is the message
				if res != nil {
					// Print each log message in the form "timestamp level message".
					// For example, "2022-06-03T00:05:10.042Z info Component keycloak successfully installed"
					fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("%s %s %s\n", res[0][2], res[0][1], res[0][4]))
				}
			} else if logFormat == cmdhelpers.LogFormatJSON {
				fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("%s\n", sc.Text()))
			}

			// Return when the Verrazzano install has completed
			vz, err := helpers.GetVerrazzanoResource(client, namespacedName)
			if err != nil {
				resChan <- err
			}
			for _, condition := range vz.Status.Conditions {
				if condition.Type == vzapi.CondInstallComplete {
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
			fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Timeout %v exceeded waiting for install to complete\n", timeout.String()))
		}
	}

	return nil
}

// validateCmd - validate the command line options
func validateCmd(cmd *cobra.Command) error {
	if cmd.PersistentFlags().Changed(constants.VersionFlag) && cmd.PersistentFlags().Changed(constants.OperatorFileFlag) {
		return fmt.Errorf("--%s and --%s cannot both be specified", constants.VersionFlag, constants.OperatorFileFlag)
	}
	return nil
}
