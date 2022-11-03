// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/explain"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/start"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/version"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

var kubeconfig string
var context string

const (
	CommandName = "psrctl"
	helpShort   = "The psrctl tool runs PSR scenarios and use cases in a Verrazzano environment"
	helpLong    = `The psrctl tool runs PSR scenarios and use cases in a Verrazzano environment.  A use case is
a unit of work executing in a pod. Some examples of use cases are: post log records to OpenSearch,
scale OpenSearch, upgrade Verrazzano, randomly terminate MySQL pods, etc.  Use cases are executed in the context
of a worker doing work in a pod running in a continuous loop.  Workers execute single task only (the use case).  
Workers can be scaled out vertically (multiple threads) and horizontally (multiple replicas).  There are 
a few configuration tuning parameters that control the execution, such as time to sleep between loop iterations,
the number of loop iterations, etc.

A scenario is set of use cases, where the use cases run in parallel, independent from one another.  For example,
a scenario might both create and get OpenSearch log records, randomly terminate OpenSearch pods, while upgrading Verrazzano.
`
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)

	// Add global flags
	cmd.PersistentFlags().StringVar(&kubeconfig, constants.GlobalFlagKubeConfig, "", constants.GlobalFlagKubeConfigHelp)
	cmd.PersistentFlags().StringVar(&context, constants.GlobalFlagContext, "", constants.GlobalFlagContextHelp)

	// Add commands
	cmd.AddCommand(explain.NewCmdExplain(vzHelper))
	cmd.AddCommand(start.NewCmdStart(vzHelper))
	cmd.AddCommand(version.NewCmdVersion(vzHelper))

	return cmd
}
