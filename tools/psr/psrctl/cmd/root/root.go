// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/explain"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/list"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/start"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/stop"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "psrctl"
	helpShort   = "The psrctl tool runs PSR scenarios in a Verrazzano environment"
	helpLong    = `The psrctl tool runs PSR scenarios in a Verrazzano environment.  
A scenario consists of a set of use cases, where the use cases run in parallel, independent from each other.
Each use case is installed as a single Helm release.
	
A use case is a specific type of work executing in a pod or set of pods. Some examples of use cases are: 
post log records to OpenSearch, scale OpenSearch, upgrade Verrazzano, randomly terminate MySQL pods, etc.  
Use cases are executed in the context of a worker doing work in a pod running in a continuous loop.  
Workers execute single task only (the use case).  Workers can be scaled out vertically (multiple threads) 
and horizontally (multiple replicas).  There are a few configuration tuning parameters that control the execution, 
such as time to sleep between loop iterations, the number of loop iterations, etc.
	
All scenario and use case configuration is controlled by YAML files that are compiled into the psrctl image.`
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)

	// Add commands
	cmd.AddCommand(explain.NewCmdExplain(vzHelper))
	cmd.AddCommand(start.NewCmdStart(vzHelper))
	cmd.AddCommand(stop.NewCmdStop(vzHelper))
	cmd.AddCommand(list.NewCmdList(vzHelper))

	return cmd
}
