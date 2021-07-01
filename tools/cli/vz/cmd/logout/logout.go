// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logout

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"strings"
)

type LogoutOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewLogoutOptions(streams genericclioptions.IOStreams) *LogoutOptions {
	return &LogoutOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdLogout(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewLogoutOptions(streams)
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "vz logout",
		Long:  "vz_logout",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := logout(); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func logout() error {
	kubeConfigLoc, err := getKubeconfigLocation()
	if err != nil {
		return err
	}
	mykubeConfig, _ := clientcmd.LoadFromFile(kubeConfigLoc)

	delete(mykubeConfig.Clusters, "verrazzano")
	delete(mykubeConfig.AuthInfos, "verrazzano")
	delete(mykubeConfig.Contexts, mykubeConfig.CurrentContext)
	mykubeConfig.CurrentContext = strings.Split(mykubeConfig.CurrentContext, "@")[1]

	err = WriteKubeConfigToDisk(kubeConfigLoc,
		mykubeConfig,
	)
	if err != nil {
		fmt.Println("Unable to write the new kubconfig to disk")
		return err
	}
	fmt.Println("Logout successful!")
	return nil
}

// Write the kubeconfig object to a file in yaml format
func WriteKubeConfigToDisk(filename string, kubeconfig *clientcmdapi.Config) error {
	err := clientcmd.WriteToFile(*kubeconfig, filename)
	if err != nil {
		return err
	}
	return nil
}

// Helper function to obtain the default kubeconfig location
func getKubeconfigLocation() (string, error) {

	var kubeconfig string
	kubeconfigEnvVar := os.Getenv("KUBECONFIG")

	if len(kubeconfigEnvVar) > 0 {
		// Find using environment variables
		kubeconfig = kubeconfigEnvVar
	} else if home := homedir.HomeDir(); home != "" {
		// Find in the ~/.kube/ directory
		kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		// give up
		return "", errors.New("Could not find kube config")
	}
	return kubeconfig, nil
}
