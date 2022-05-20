// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"io"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/spf13/cobra"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RootCmdContext struct {
	genericclioptions.IOStreams
}

// GetOutputStream - return the output stream
func (rc *RootCmdContext) GetOutputStream() io.Writer {
	return rc.IOStreams.Out
}

// GetErrorStream - return the error stream
func (rc *RootCmdContext) GetErrorStream() io.Writer {
	return rc.IOStreams.ErrOut
}

// GetClient - return a kubernetes client that supports the schemes used by the CLI
func (rc *RootCmdContext) GetClient(cmd *cobra.Command) (client.Client, error) {
	config, err := getConfig(cmd)
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

// NewRootCmdContext - create the root command context object
func NewRootCmdContext(streams genericclioptions.IOStreams) *RootCmdContext {
	return &RootCmdContext{
		IOStreams: streams,
	}
}

// getConfig - create config attributes for Kubernetes client
func getConfig(cmd *cobra.Command) (*rest.Config, error) {
	// Determine the location of the kube config.  It was either specified on the command line
	// or we find the default location.
	kubeConfigLoc, err := cmd.Flags().GetString(constants.GlobalFlagKubeConfig)
	if err != nil {
		return nil, err
	}
	if len(kubeConfigLoc) == 0 {
		kubeConfigLoc, err = k8sutil.GetKubeConfigLocation()
		if err != nil {
			return nil, err
		}
	}

	// Was a kube config context specified on the command line?
	context, err := cmd.Flags().GetString(constants.GlobalFlagContext)
	if err != nil {
		return nil, err
	}

	// Create the config
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigLoc},
		&clientcmd.ConfigOverrides{CurrentContext: context}).ClientConfig()
}
