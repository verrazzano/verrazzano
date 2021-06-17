// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterManifestOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
	outputOptions string
}

func NewClusterManifestOptions (streams genericclioptions.IOStreams) *ClusterManifestOptions {
	return &ClusterManifestOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterManifest (streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewClusterManifestOptions(streams)
	cmd := &cobra.Command{
		Use:   "get-registration-manifest [name]",
		Short: "Get the registration manifest for the manged cluster",
		Long:  "Get the registration manifest to be applied on the managed cluster",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.args = args
			if err := o.getManifest(kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&o.outputOptions, "output", "o", "yaml", helpers.OutputUsage)
	return cmd
}

func (o *ClusterManifestOptions) getManifest(kubenetesInterface helpers.Kubernetes) error {
	//name of managedCluster
	vmcName := o.args[0]

	//get the vmcObject to get the name of the manifest secret
	clientset, err := kubenetesInterface.NewClientSet()

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
	kclientset := kubenetesInterface.NewKubernetesClientSet()
	secret, err := kclientset.CoreV1().Secrets(vmcNamespace).Get(context.Background(), manifestName, v1.GetOptions{})

	if err != nil {
		return err
	}

	//err = helpers.PrintJsonYaml("yaml", secret.Data["yaml"], o.Out)
	_, err = fmt.Fprintln(o.Out, string(secret.Data["yaml"]))
	return err
}

