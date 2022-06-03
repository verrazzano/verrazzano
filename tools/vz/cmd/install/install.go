// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	vzconstants "github.com/verrazzano/verrazzano/pkg/constants"
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
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CommandName = "install"
	helpShort   = "Install Verrazzano"
	helpLong    = `Install the Verrazzano Platform Operator and install the Verrazzano components specified by the Verrazzano CR provided on the command line.`
	helpExample = `
# Install the latest version of Verrazzano using the prod profile. Stream the logs to the console until the install completes.
vz install

# Install version 1.3.0 using a dev profile, timeout the command after 20 minutes.
vz install --version v1.3.0 --set profile=dev --timeout 20m

# Install version 1.3.0 using a dev profile with elasticsearch disabled and wait for the install to complete.
vz install --version v1.3.0 --set profile=dev --set components.elasticsearch.enabled=false

# Install the latest version of Verrazzano using CR overlays and explicit value sets.  Output the logs in json format.
vz install -f base.yaml -f custom.yaml --set profile=prod --log-format json`

	verrazzanoPlatformOperator     = "verrazzano-platform-operator"
	verrazzanoPlatformOperatorWait = 1
)

var logsEnum = cmdhelpers.LogsFormatSimple

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
	// Get the kubernetes clientset.  This will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Apply the Verrazzano operator.yaml.
	err = applyPlatformOperatorYaml(cmd, vzHelper)
	if err != nil {
		return err
	}

	// Get the controller runtime client
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Wait for the platform operator to be ready before we create the Verrazzano resource.
	podName, err := waitForPlatformOperator(client, vzHelper)
	if err != nil {
		return err
	}

	// Create the Verrazzano install resource.
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	err = client.Create(context.TODO(), vz)
	if err != nil {
		return err
	}

	// Wait for the Verrazzano install to complete
	return waitForInstallToComplete(kubeClient, cmd, vzHelper, podName)
}

// applyPlatformOperatorYaml applies a given version of the platform operator yaml file
func applyPlatformOperatorYaml(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	// Get the version from the command line
	version, err := cmd.PersistentFlags().GetString(constants.VersionFlag)
	if err != nil {
		return err
	}
	if version == constants.VersionFlagDefault {
		// Find the latest release version of Verrazzano
		version, err = helpers.GetLatestReleaseVersion()
		if err != nil {
			return err
		}
	}

	// Apply the Verrazzano operator.yaml. A valid version must be specified for this to succeed.
	kubectl := exec.Command("kubectl", "apply", "-f", fmt.Sprintf("https://github.com/verrazzano/verrazzano/releases/download/%s/operator.yaml", version))
	var stdout bytes.Buffer
	kubectl.Stdout = &stdout
	var stderr bytes.Buffer
	kubectl.Stderr = &stderr
	cmdErr := kubectl.Run()
	if cmdErr != nil {
		return fmt.Errorf("Failed to download operator.yaml: %s", stderr.String())
	}
	fmt.Fprintf(vzHelper.GetOutputStream(), stdout.String())

	return nil
}

// waitForPlatformOperator waits for the verrazzano-platform-operator to be ready
func waitForPlatformOperator(client clipkg.Client, vzHelper helpers.VZHelper) (string, error) {
	// Find the verrazzano-platform-operator using the app label selector
	appLabel, _ := labels.NewRequirement("app", selection.Equals, []string{verrazzanoPlatformOperator})
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
		return "", fmt.Errorf("Failed to list pods %v", err)
	}
	if len(podList.Items) == 0 {
		return "", fmt.Errorf("%s pod not found in namespace %s", verrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	}
	if len(podList.Items) > 1 {
		return "", fmt.Errorf("More than one %s pod was found in namespace %s", verrazzanoPlatformOperator, vzconstants.VerrazzanoInstallNamespace)
	}

	// We found the verrazzano-platform-operator pod. Wait until it's containers are ready.
	pod := &corev1.Pod{}
	seconds := 0
	for {
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

		time.Sleep(verrazzanoPlatformOperatorWait * time.Second)
		seconds += verrazzanoPlatformOperatorWait
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("\rWaiting for verrazzano-platform-operator to be ready - %d seconds", seconds))
	}
	return pod.Name, nil
}

// waitForInstallToComplete waits for the Verrazzano install to complete and shows the logs of
// the ongoing Verrazzano install.
func waitForInstallToComplete(kubeClient *kubernetes.Clientset, cmd *cobra.Command, vzHelper helpers.VZHelper, podName string) error {
	// Tail the log messages starting at the current time.
	sinceTime := metav1.Now()
	rc, err := kubeClient.CoreV1().Pods(vzconstants.VerrazzanoInstallNamespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: verrazzanoPlatformOperator,
		Follow:    true,
		SinceTime: &sinceTime,
	}).Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("Failed to get logs stream: %v", err)
	}
	defer rc.Close()

	sc := bufio.NewScanner(rc)
	sc.Split(bufio.ScanLines)
	for sc.Scan() {
		re := regexp.MustCompile(`"level":"(.*?)","@timestamp":"(.*?)","caller":"(.*?)","message":"(.*?)",`)
		res := re.FindAllStringSubmatch(sc.Text(), -1)
		// res[0][2] is the timestamp
		// res[0][1] is the level
		// res[0][4] is the message
		if res != nil {
			// Print each log message in the form "timestamp level message".
			// For example, "2022-06-03T00:05:10.042Z info Component keycloak successfully installed"
			fmt.Fprintln(vzHelper.GetOutputStream(), fmt.Sprintf("%s %s %s", res[0][2], res[0][1], res[0][4]))
			// Return when the Verrazzano install has completed
			if strings.Compare(res[0][4], "Successfully installed Verrazzano") == 0 {
				return nil
			}
		}
	}

	return nil
}
