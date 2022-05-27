// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// GlobalFlagKubeConfig - global flag for specifying the location of the kube config
const GlobalFlagKubeConfig = "kubeconfig"

// GlobalFlagContext - global flag for specifying which kube config context to use
const GlobalFlagContext = "context"

// GlobalFlagHelp - global help flag
const GlobalFlagHelp = "help"

// Flags that are common to more than one command
const (
	WaitFlag     = "wait"
	WaitFlagHelp = "Wait for the command to complete."

	TimeoutFlag     = "timeout"
	TimeoutFlagHelp = "Time to wait for a command to complete"

	VersionFlag     = "version"
	VersionFlagHelp = "The version of Verrazzano to install or upgrade"

	DryRunFlag = "dry-run"

	OperatorFileFlag     = "operator-file"
	OperatorFileFlagHelp = "The path to the file for installing the Verrazzano platform operator. The default is derived from the version string."

	LogsFlag     = "logs"
	LogsFlagHelp = "Print the logs until the command completes. Valid output formats are \"pretty\" and \"json\""

	FilenameFlag     = "filename"
	FilenameFlagHelp = "Path to file containing Verrazzano custom resource.  This flag can be specified multiple times to overlay multiple files."
)
