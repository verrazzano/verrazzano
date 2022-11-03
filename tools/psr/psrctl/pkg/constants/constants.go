// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

// GlobalFlagKubeConfig - global flag for specifying the location of the kube config
const GlobalFlagKubeConfig = "kubeconfig"
const GlobalFlagKubeConfigHelp = "Path to the kubeconfig file to use"

// GlobalFlagContext - global flag for specifying which kube config context to use
const GlobalFlagContext = "context"
const GlobalFlagContextHelp = "The name of the kubeconfig context to use"

// GlobalFlagHelp - global help flag
const GlobalFlagHelp = "help"

// Flags that are common to more than one command
const (
	WaitFlag        = "wait"
	WaitFlagHelp    = "Wait for the command to complete and stream the logs to the console. The wait period is controlled by --timeout."
	WaitFlagDefault = true

	TimeoutFlag     = "timeout"
	TimeoutFlagHelp = "Limits the amount of time a command will wait to complete"

	VPOTimeoutFlag     = "platform-operator-timeout"
	VPOTimeoutFlagHelp = "Limits the amount of time a command will wait for the Verrazzano Platform Operator to be ready"

	VersionFlag            = "version"
	VersionFlagDefault     = "latest"
	VersionFlagInstallHelp = "The version of Verrazzano to install"
	VersionFlagUpgradeHelp = "The version of Verrazzano to upgrade to"

	DryRunFlag = "dry-run"

	SetFlag          = "set"
	SetFlagShorthand = "s"
	SetFlagHelp      = "Override a Verrazzano resource value (e.g. --set profile=dev).  This flag can be specified multiple times."

	OperatorFileFlag     = "operator-file"
	OperatorFileFlagHelp = "The path to the file for installing the Verrazzano platform operator. The default is derived from the version string."

	LogFormatFlag = "log-format"
	LogFormatHelp = "The format of the log output. Valid output formats are \"simple\" and \"json\"."

	FilenameFlag          = "filename"
	FilenameFlagShorthand = "f"
	FilenameFlagHelp      = "Path to file containing Verrazzano custom resource.  This flag can be specified multiple times to overlay multiple files.  Specifying \"-\" as the filename accepts input from stdin."

	VerboseFlag          = "verbose"
	VerboseFlagShorthand = "v"
	VerboseFlagDefault   = false
	VerboseFlagUsage     = "Enable verbose output."
)
