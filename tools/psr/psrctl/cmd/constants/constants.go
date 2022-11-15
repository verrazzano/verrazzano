// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	FlagScenario       = "scenario"
	FlagsScenarioShort = "s"
	FlagScenarioHelp   = "the scenario ID"

	FlagNamespace      = "namespace"
	FlagNamespaceShort = "n"
	FlagNamespaceHelp  = "the namespace for the scenario"

	FlagVerbose      = "verbose"
	FlagVerboseShort = "v"
	FlagVerboseHelp  = "verbose output"

	FlagAll      = "all-namespaces"
	FlagAllShort = "A"
	FlagAllHelp  = "all namespaces"

	FlagScenarioDir      = "scenario-directory"
	FlagScenarioDirShort = "d"
	FlagScenarioDirHelp  = `a directory that contains a scenario directory at any level in the directory tree.  This allows you to run scenarios that are not compiled into the psrctl binary.`
)
