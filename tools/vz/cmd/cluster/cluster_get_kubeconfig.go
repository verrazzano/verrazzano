// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/capi"
	os2 "github.com/verrazzano/verrazzano/pkg/os"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	getKubeconfigSubCommandName = "get-kubeconfig"
	getKubeconfigHelpShort      = "Verrazzano cluster get-kubeconfig"
	getKubeconfigHelpLong       = `The command 'cluster get-kubeconfig' gets the kubeconfig for the cluster with the given name and saves it to the specified file (defaults to "` + constants.ClusterNameFlagDefault + `")`
	getKubeconfigHelpExample    = `vz cluster get-kubeconfig --name mycluster --path path_to_my_file`
)

func newSubcmdGetKubeconfig(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, getKubeconfigSubCommandName, getKubeconfigHelpShort, getKubeconfigHelpLong)
	cmd.Example = getKubeconfigHelpExample
	cmd.PersistentFlags().String(constants.ClusterNameFlagName, constants.ClusterNameFlagDefault, constants.ClusterNameFlagHelp)
	cmd.PersistentFlags().String(constants.KubeconfigPathFlagName, constants.KubeconfigPathFlagDefault, constants.KubeconfigPathFlagHelp)

	// add a hidden cluster type flag for testing purposes, even though get-kubeconfig does not require it, with an empty default so that
	// the underlying CAPI default is used if unspecified
	cmd.PersistentFlags().String(constants.ClusterTypeFlagName, "", constants.ClusterTypeFlagHelp)
	cmd.PersistentFlags().MarkHidden(constants.ClusterTypeFlagName)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdClusterGetKubeconfig(vzHelper, cmd, args)
	}

	return cmd
}

func runCmdClusterGetKubeconfig(helper helpers.VZHelper, cmd *cobra.Command, args []string) error {
	clusterName, err := cmd.PersistentFlags().GetString(constants.ClusterNameFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterNameFlagName, err)
	}

	filePath, err := cmd.PersistentFlags().GetString(constants.KubeconfigPathFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.KubeconfigPathFlagName, err)
	}

	clusterType, err := cmd.PersistentFlags().GetString(constants.ClusterTypeFlagName)
	if err != nil {
		return fmt.Errorf("Failed to get the %s flag: %v", constants.ClusterTypeFlagName, err)
	}

	if filePath == "" {
		filePath, err = defaultKubeconfigFilePath()
		if err != nil {
			return err
		}
		fmt.Fprintf(helper.GetOutputStream(), "No path specified - using %s\n", filePath)
	}

	cluster, err := capi.NewBoostrapCluster(capi.ClusterConfig{
		ClusterName: clusterName,
		Type:        clusterType,
	})
	if err != nil {
		return err
	}

	exists, err := os2.FileExists(filePath)
	if err != nil {
		return err
	}
	existingKubeconfig := false
	newKubeconfigPath := filePath
	if exists {
		fmt.Fprintf(helper.GetOutputStream(), "the file %s already exists - the kubeconfig for cluster %s will be merged into it", filePath, clusterName)
		existingKubeconfig = true
		newKubeconfigFile, err := os2.CreateTempFile(generateTempKubeconfigFilename(), []byte{})
		if err != nil {
			return err
		}
		newKubeconfigPath = newKubeconfigFile.Name()
		defer func() {
			os.Remove(newKubeconfigPath)
		}()
	}

	kubeconfigContents, err := cluster.GetKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to get the kubeconfig for cluster %s: %v", clusterName, err)
	}

	message := "Wrote kubeconfig to file"
	if existingKubeconfig {
		// write new kubeconfig to temp file and merge with existing file
		err = os.WriteFile(newKubeconfigPath, []byte(kubeconfigContents), 0700)
		kubeconfigContents, err = mergeKubeconfigs(filePath, newKubeconfigPath)
		message = "Merged kubeconfig into existing file"
	}
	if err = os.WriteFile(filePath, []byte(kubeconfigContents), 0700); err != nil {
		fmt.Fprintf(helper.GetOutputStream(), "Failed to write kubeconfig to file %s - %v\n", filePath, err)
		return err
	}
	fmt.Fprintf(helper.GetOutputStream(), "%s %s\n", message, filePath)
	return nil
}

func generateTempKubeconfigFilename() string {
	return fmt.Sprintf("tempkubeconfig_%d", time.Now().UnixNano())
}

func defaultKubeconfigFilePath() (string, error) {
	kubePath := os.Getenv("KUBECONFIG")
	if kubePath != "" {
		return kubePath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return path.Join(home, ".kube", "config"), nil
}

func mergeKubeconfigs(existingConfigPath string, newConfigPath string) (string, error) {
	loadingRules := clientcmd.ClientConfigLoadingRules{
		Precedence: []string{existingConfigPath, newConfigPath}}
	mergedConfig, err := loadingRules.Load()
	if err != nil {
		return "", fmt.Errorf("failed to merge kubeconfigs: %v", err)
	}
	mergedBytes, err := clientcmd.Write(*mergedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to write merged kubeconfig: %v", err)
	}
	return string(mergedBytes), nil
}
