// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/spf13/pflag"
	"github.com/verrazzano/verrazzano/tools/vz/cmd"
)

func main() {
	flags := pflag.NewFlagSet("vz", pflag.ExitOnError)
	pflag.CommandLine = flags

	rootCmd := cmd.NewRootCmd(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
