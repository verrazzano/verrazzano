// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmd

import (
	"io"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/spf13/cobra"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/status"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var kubeconfig string
var context string

const (
	GlobalFlagKubeconfig = "kubeconfig"
	GlobalFlagContext    = "context"
)

type RootCmdContext struct {
	genericclioptions.IOStreams
}

// GetOutputStream - return the output stream
func (rc *RootCmdContext) GetOutputStream() io.Writer {
	return rc.IOStreams.Out
}

// GetClient - return a kubernetes client that supports the schemes used by the CLI
func (rc *RootCmdContext) GetClient() (client.Client, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	_ = vzapi.AddToScheme(scheme)
	_ = clustersv1alpha1.AddToScheme(scheme)
	_ = platformopclusters.AddToScheme(scheme)
	_ = oamv1alpha2.SchemeBuilder.AddToScheme(scheme)
	_ = corev1.SchemeBuilder.AddToScheme(scheme)

	return client.New(config, client.Options{Scheme: scheme})
}

func newRootCmdContext(streams genericclioptions.IOStreams) *RootCmdContext {
	return &RootCmdContext{
		IOStreams: streams,
	}
}

func NewRootCmd(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vz",
		Short: "Verrazzano CLI",
		Long:  "Verrazzano CLI",
	}

	// Add global flags
	cmd.PersistentFlags().StringVarP(&kubeconfig, GlobalFlagKubeconfig, "c", "", "Kubernetes configuration file")
	cmd.PersistentFlags().StringVar(&context, GlobalFlagContext, "", "The name of the kubeconfig context to use")

	// Add commands
	rc := newRootCmdContext(streams)
	cmd.AddCommand(status.NewCmdStatus(rc))
	cmd.AddCommand(version.NewCmdVersion())

	return cmd
}
