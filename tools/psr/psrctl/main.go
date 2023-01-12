// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/root"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	// Extract the manifests and write them to a temp directory
	// This sets the global embedded.Manifests var
	err := manifest.InitGlobalManifests()
	if err != nil {
		fmt.Printf("Failed to initial manifests %v", err)
		os.Exit(1)
	}
	defer manifest.CleanupManifests()

	flags := pflag.NewFlagSet("psrctl", pflag.ExitOnError)
	pflag.CommandLine = flags

	rc := helpers.NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	rootCmd := root.NewRootCmd(rc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
