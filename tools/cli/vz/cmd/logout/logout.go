// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logout

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
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

	// Obtain the default kubeconfig's location
	kubeConfigLoc,err := helpers.GetKubeconfigLocation()
	if err!=nil {
		return err
	}

	// Load the default kubeconfig's configuration into clientcmdapi object
	mykubeConfig, _ := clientcmd.LoadFromFile(kubeConfigLoc)
	if err!=nil {
		fmt.Println("Unable to load kubeconfig, check permissions")
		return err
	}

	// Remove the cluster with nickname verrazzano
	helpers.RemoveCluster(mykubeConfig, "verrazzano")

	// Remove the user with nickname verrazzano
	helpers.RemoveUser(mykubeConfig,"verrazzano")

	// Remove the currentcontext
	helpers.RemoveContext(mykubeConfig, mykubeConfig.CurrentContext)

	// Set currentcluster to the cluster before the user logged in
	helpers.SetCurrentContext(mykubeConfig, strings.Split(mykubeConfig.CurrentContext,"@")[1])

	// Write kubeconfig to file
	err = clientcmd.WriteToFile(*mykubeConfig,
				     kubeConfigLoc,
				  )
	if err!=nil {
		fmt.Println("Unable to write the new kubconfig to disk")
		return err
	}
	fmt.Println("Logout successful!")
	return nil
}
