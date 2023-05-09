// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import (
	"os"

	"github.com/spf13/pflag"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/root"
	"github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("vcm", pflag.ExitOnError)
	pflag.CommandLine = flags

	rc := helpers.NewRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	rootCmd := root.NewRootCmd(rc, nil, nil)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
