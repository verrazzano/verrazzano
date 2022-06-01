// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"io"

	"github.com/verrazzano/verrazzano/pkg/semver"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/spf13/cobra"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	platformopclusters "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/github"
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

// GetInputStream - return the input stream
func (rc *RootCmdContext) GetInputStream() io.Reader {
	return rc.IOStreams.In
}

// GetClient - return a kubernetes client that supports the schemes used by the CLI
func (rc *RootCmdContext) GetClient(cmd *cobra.Command) (client.Client, error) {
	// Get command line value of kubeConfig location
	kubeConfigLoc, err := cmd.Flags().GetString(constants.GlobalFlagKubeConfig)
	if err != nil {
		return nil, err
	}

	// Get command line value of kubeContext
	context, err := cmd.Flags().GetString(constants.GlobalFlagContext)
	if err != nil {
		return nil, err
	}

	config, err := k8sutil.GetKubeConfigGivenPathAndContext(kubeConfigLoc, context)
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

// GetLatestReleaseVersion - get the version of the latest release of Verrazzano
func (rc *RootCmdContext) GetLatestReleaseVersion() (string, error) {
	// Get the list of all Verrazzano releases
	releases, err := github.ListReleases()
	if err != nil {
		return "", fmt.Errorf("Failed to get list of Verrazzano releases: %s", err.Error())
	}

	// Determine which tag is the latest release
	var latestRelease *semver.SemVersion
	for _, tag := range releases {
		tagSemver, err := semver.NewSemVersion(tag)
		if err != nil {
			return "", err
		}
		if latestRelease == nil {
			latestRelease = tagSemver
		} else {
			if tagSemver.IsGreatherThan(latestRelease) {
				latestRelease = tagSemver
			}
		}
	}
	return fmt.Sprintf("v%s", latestRelease.ToString()), nil
}

// NewRootCmdContext - create the root command context object
func NewRootCmdContext(streams genericclioptions.IOStreams) *RootCmdContext {
	return &RootCmdContext{
		IOStreams: streams,
	}
}
