// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"io"
	"net/http"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
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

// GetInputStream - return the input stream
func (rc *RootCmdContext) GetInputStream() io.Reader {
	return rc.IOStreams.In
}

// GetClient - return a Kubernetes controller runtime client that supports the schemes used by the CLI
func (rc *RootCmdContext) GetClient(cmd *cobra.Command) (client.Client, error) {
	config, err := getKubeConfigGivenCommand(cmd)
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{Scheme: helpers.NewScheme()})
}

// GetKubeClient - return a Kubernetes clientset for use with the go-client
func (rc *RootCmdContext) GetKubeClient(cmd *cobra.Command) (kubernetes.Interface, error) {
	config, err := getKubeConfigGivenCommand(cmd)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// GetDynamicClient - return a dynamic clientset for use with the go-client
func (rc *RootCmdContext) GetDynamicClient(cmd *cobra.Command) (dynamic.Interface, error) {
	config, err := getKubeConfigGivenCommand(cmd)
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
}

func getKubeConfigGivenCommand(cmd *cobra.Command) (*rest.Config, error) {
	// Get command line value of --kubeconfig
	kubeConfigLoc, err := cmd.Flags().GetString(constants.GlobalFlagKubeConfig)
	if err != nil {
		return nil, err
	}

	// Get command line value of --context
	context, err := cmd.Flags().GetString(constants.GlobalFlagContext)
	if err != nil {
		return nil, err
	}

	config, err := k8sutil.GetKubeConfigGivenPathAndContext(kubeConfigLoc, context)
	if err != nil {
		return nil, err
	}
	return config, err
}

// GetHTTPClient - return an HTTP client
func (rc *RootCmdContext) GetHTTPClient() *http.Client {
	return &http.Client{}
}

// NewRootCmdContext - create the root command context object
func NewRootCmdContext(streams genericclioptions.IOStreams) *RootCmdContext {
	return &RootCmdContext{
		IOStreams: streams,
	}
}
