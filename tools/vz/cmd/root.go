// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	v8oclientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
)

var kubeconfig string
var context string

const (
	GlobalFlagKubeconfig = "kubeconfig"
	GlobalFlagContext    = "context"
)

type RootContext struct {
}

func (rc *RootContext) NewVerrazzanoClientSet() (v8oclientset.Interface, error) {
	var client v8oclientset.Interface
	kubeConfig, err := k8sutil.GetKubeConfig()
	if err != nil {
		return client, err
	}
	client, err = v8oclientset.NewForConfig(kubeConfig)
	return client, err
}

func NewRootContext() *RootContext {
	return &RootContext{}
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vz",
		Short: "Verrazzano CLI",
		Long:  "Verrazzano CLI",
	}

	// Add global flags
	cmd.PersistentFlags().StringVarP(&kubeconfig, GlobalFlagKubeconfig, "c", "", "Kubernetes configuration file")
	cmd.PersistentFlags().StringVar(&context, GlobalFlagContext, "", "The name of the kubeconfig context to use")

	// Add commands
	rc := NewRootContext()
	cmd.AddCommand(status.NewCmdStatus(rc))
	cmd.AddCommand(version.NewCmdVersion())

	return cmd
}
