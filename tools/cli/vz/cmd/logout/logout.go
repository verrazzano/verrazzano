// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logout

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// Struct to store Logout-command related data. eg.flags,streams,args..
type LogoutOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

// Creates a LogoutOptions struct to run the logout command
func NewLogoutOptions(streams genericclioptions.IOStreams) *LogoutOptions {
	return &LogoutOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

// Calls the logout function to complete logout
func NewCmdLogout(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewLogoutOptions(streams)
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "vz logout",
		Long:  "vz_logout",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := logout(streams); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func logout(streams genericclioptions.IOStreams) error {
	// Check if the user is already logged out
	isLoggedOut, err := helpers.LoggedOut()
	if err != nil {
		return err
	}
	if isLoggedOut {
		_, err := fmt.Fprintln(streams.Out, "Already Logged out")
		return err
	}

	// Remove all the stored auth data
	err = helpers.RemoveAllAuthData()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(streams.Out, "Logout successful!")
	return err
}
