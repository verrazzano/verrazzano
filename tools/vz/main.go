// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"os"

	"github.com/verrazzano/verrazzano/tools/vz/cmd"

	"github.com/verrazzano/verrazzano/tools/vz/cmd/root"

	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("vz", pflag.ExitOnError)
	pflag.CommandLine = flags

	rc := cmd.NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	rootCmd := root.NewRootCmd(rc)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
