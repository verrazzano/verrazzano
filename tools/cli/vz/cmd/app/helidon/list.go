// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidon

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	v1alpha12 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	clustersclient "github.com/verrazzano/verrazzano/application-operator/clients/clusters/clientset/versioned/typed/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/clients/oam/clientset/versioned/typed/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tools/cli/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var project string

type HelidonListOptions struct {
	configFlags *genericclioptions.ConfigFlags
	args        []string
	genericclioptions.IOStreams
}

func NewHelidonListOptions(streams genericclioptions.IOStreams) *HelidonListOptions {
	return &HelidonListOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
}

func NewCmdAppHelidonList(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewHelidonListOptions(streams)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Helidon applications",
		Long:  "List Helidon applications",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := listHelidonApplications(cmd, args); err != nil {
				return err
			}
			return nil
		},
	}
	o.configFlags.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&project, "project", "p", "", "Project name")
	return cmd
}

func listHelidonApplications(cmd *cobra.Command, args []string) error {
	// connect to the server
	config := pkg.GetKubeConfig()
	oamclientset, err := v1alpha1.NewForConfig(config)
	if err != nil {
		fmt.Print("could not get the OAM/Helidon clientset")
	}

	// get a list of namespaces for the given project
	clientset, err := clustersclient.NewForConfig(config)
	if err != nil {
		fmt.Print("could not get the clusters clientset")
	}

	// get a list of the projects
	p, err := clientset.VerrazzanoProjects("verrazzano-mc").Get(context.Background(), project, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if p == nil {
		return errors.New("no such project found")
	}

	namespaces := func() []string {
		result := []string{}
		for _, x := range p.Spec.Template.Namespaces {
			result = append(result, x.Metadata.Name)
		}
		return result
	}()

	// get a list of the helidon applications
	apps := []v1alpha12.VerrazzanoHelidonWorkload{}

	for _, n := range namespaces {
		w, err := oamclientset.VerrazzanoHelidonWorkloads(n).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return err
		}
		apps = append(apps, w.Items...)
	}

	// check if the list is empty
	if len(apps) == 0 {
		fmt.Println(helpers.NothingFound)
		return nil
	}

	// print out details of the projects
	headings := []string{"NAMESPACE", "NAME", "AGE", "HOSTNAME"}
	data := [][]string{}
	for _, app := range apps {
		rowData := []string{
			app.Namespace,
			app.Name,
			helpers.Age(app.CreationTimestamp),
			pkg.GetHostnameFromGateway(app.Namespace, app.Name+"-appconf"),
		}
		data = append(data, rowData)
	}

	// print out the data
	helpers.PrintTable(headings, data, cmd.OutOrStdout())
	return nil
}
