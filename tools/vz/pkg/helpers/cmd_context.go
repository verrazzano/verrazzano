// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"io"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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

func NewRootCmdContext(streams genericclioptions.IOStreams) *RootCmdContext {
	return &RootCmdContext{
		IOStreams: streams,
	}
}
