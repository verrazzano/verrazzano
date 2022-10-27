// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/verrazzano/verrazzano/tools/psr/cli/cmd/root"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("psrctl", pflag.ExitOnError)
	pflag.CommandLine = flags

	rc := helpers.NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	rootCmd := root.NewRootCmd(rc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
