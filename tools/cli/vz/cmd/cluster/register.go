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
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ClusterRegisterOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams

	caSecret    string
	description string
}

func NewClusterRegisterOptions(streams genericclioptions.IOStreams) *ClusterRegisterOptions {
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.args = args
			if err := o.registerCluster(kubernetesInterface); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&o.description, "description", "d", "", "Description of the managed cluster")
	cmd.Flags().StringVarP(&o.caSecret, "casecret", "c", "", "Name of the CA Secret")
	return cmd
}

func (o *ClusterRegisterOptions) registerCluster(kubernetesInterface helpers.Kubernetes) error {
	// Name of the managedCluster
	mcName := o.args[0]

	// Check for verrazzano-admin-cluster configmap
	// If doesn't exist, create it
	if err := o.checkConfigMap(kubernetesInterface); err != nil {
		return err
	}

	// CA Secret name was not provided
	if len(o.caSecret) == 0 {
		return errors.New("CA secret is needed")
	}

	// Create the vmc resource for the managed cluster
	vmcObject := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mcName,
			Namespace: vmcNamespace,
		},
		Spec: v1alpha1.VerrazzanoManagedClusterSpec{
			Description: o.description,
			CASecret:    o.caSecret,
		},
	}

	clientset, err := kubernetesInterface.NewClustersClientSet()

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

func (o *ClusterRegisterOptions) checkConfigMap(kubernetesInterface helpers.Kubernetes) error {
	client, err := kubernetesInterface.NewClientSet()
	if err != nil {
		return err
	}
	name := "verrazzano-admin-cluster"

	_, err = client.CoreV1().ConfigMaps(vmcNamespace).Get(context.Background(), name, metav1.GetOptions{})

	// Config map doesn't exist, crete one
	if err != nil && k8serror.IsNotFound(err) {
		_, err := fmt.Fprintln(o.Out, "configmap/verrazzano-admin-cluster doesn't exist\ncreating configmap/verrazzano-admin-cluster")
		if err != nil {
			return err
		}
		kubeConfig, err := kubernetesInterface.GetKubeConfig()
		if err != nil {
			return err
		}
		configMap := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: vmcNamespace,
			},
			Data: map[string]string{
				"server": kubeConfig.Host,
			},
		}
		_, err = client.CoreV1().ConfigMaps(vmcNamespace).Create(context.Background(), configMap, metav1.CreateOptions{})

		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(o.Out, "configmap/verrazzano-admin-cluster created")
		return err
	}

	return err
}
