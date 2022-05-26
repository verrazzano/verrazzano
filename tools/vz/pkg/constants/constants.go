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

const WaitFlag = "wait"
const WaitFlagHelp = "Wait for the command to complete.  It will wait for as long as --timeout."

const TimeoutFlag = "timeout"
const TimeoutFlagHelp = "Time to wait for a command to complete"

const VersionFlag = "version"
const VersionFlagHelp = "The version of Verrazzano to install or upgrade"

const DryRunFlag = "dry-run"

const OperatorFileFlag = "operator-file"
const OperatorFileFlagHelp = "The path to the file for installing the Verrazzano platform operator. The default value is derived from the version string."
