// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	cluster_client "github.com/verrazzano/verrazzano/platform-operator/clients/clusters/clientset/versioned"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterManifestOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewClusterManifestOptions (streams genericclioptions.IOStreams) *ClusterManifestOptions {
	return &ClusterManifestOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterManifest (streams genericclioptions.IOStreams) *cobra.Command {
	o := NewClusterManifestOptions(streams)
	cmd := &cobra.Command{
		Use:   "get-registration-manifest [name]",
		Short: "Get the registration manifest for the manged cluster",
		Long:  "Get the registration manifest to be applied on the managed cluster",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := getManifest(cmd, args); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func getManifest(cmd *cobra.Command, args []string) error {
	//name of managedCluster
	vmcName := args[0]

	//get the vmcObject to get the name of the manifest secret
	config := pkg.GetKubeConfig()
	clientset, err := cluster_client.NewForConfig(config)

	if err != nil {
		return nil
	}

	vmcObject, err := clientset.ClustersV1alpha1().VerrazzanoManagedClusters(vmcNamespace).Get(context.Background(), vmcName, v1.GetOptions{})

	if err != nil {
		return err
	}

	//name of the manifest secret to be applied on the managed cluster
	manifestName := vmcObject.Spec.ManagedClusterManifestSecret

	//k8s client set to fetch the secret
	kclientset := pkg.GetKubernetesClientset()
	secret, err := kclientset.CoreV1().Secrets(vmcNamespace).Get(context.Background(), manifestName, v1.GetOptions{})

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(secret.Data["yaml"]))
	return err
}

