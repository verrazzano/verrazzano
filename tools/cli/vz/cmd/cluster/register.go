// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterRegisterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams

	caSecret string
	description string
}

func NewClusterRegisterOptions (streams genericclioptions.IOStreams) *ClusterRegisterOptions {
	return &ClusterRegisterOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdClusterRegister(streams genericclioptions.IOStreams, kubernetesInterface helpers.Kubernetes) *cobra.Command {
	o := NewClusterRegisterOptions(streams)
	cmd := &cobra.Command{
		Use:   "register [name]",
		Short: "Register a new managed cluster",
		Long:  "Register a new managed cluster",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.args = args
			if err := o.registerCluster(kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	//o.configFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&o.description, "description", "d", "", "Description of the managed cluster")
	cmd.Flags().StringVarP(&o.caSecret, "casecret", "c", "", "Name of the CA Secret")
	return cmd
}

func (o *ClusterRegisterOptions) registerCluster(kubernetesInterface helpers.Kubernetes) error {
	//name of the managedCluster
	mcName := o.args[0]

	//CA Secret name was not provided
	if len(o.caSecret) == 0 {
		return errors.New("CA secret is needed")
	}

	//create the vmc resource for the managed cluster
	vmcObject := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: mcName,
			Namespace: vmcNamespace,
		},
		Spec: v1alpha1.VerrazzanoManagedClusterSpec{
			Description: o.description,
			CASecret: o.caSecret,
		},
	}

	clientset, err := kubernetesInterface.NewClientSet()

	if err != nil {
		return err
	}

	_, err = clientset.ClustersV1alpha1().VerrazzanoManagedClusters(vmcNamespace).Create(context.Background(), &vmcObject, metav1.CreateOptions{})

	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(o.Out, "verrazzanomanagedcluster/"+mcName+" created")

	if err != nil {
		return err
	}

	return nil
}
