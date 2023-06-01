// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const (
	FlagChartName            = "chart"
	FlagChartShorthand       = "c"
	FlagChartUsage           = "Name of the chart."
	FlagChartExampleKeycloak = "keycloakx"

	FlagVersionName       = "version"
	FlagVersionShorthand  = "v"
	FlagVersionUsage      = "Version of the chart."
	FlagVersionExample210 = "2.1.0"

	FlagRepoName               = "repo"
	FlagRepoShorthand          = "r"
	FlagRepoUsage              = "URL of the helm repo."
	FlagRepoExampleCodecentric = "https://codecentric.github.io/helm-charts"

	FlagDirName         = "charts-dir"
	FlagDirShorthand    = "d"
	FlagDirUsage        = "Location of charts directory."
	FlagDirExampleLocal = "./charts"

	FlagTargetVersionName       = "target-version"
	FlagTargetVersionShorthand  = "t"
	FlagTargetVersionUsage      = "Target downstream version of the chart."
	FlagTargetVersionExample002 = "0.0.2"

	FlagUpstreamProvenanceName      = "upstream-provenance"
	FlagUpstreamProvenanceShorthand = "u"
	FlagUpstreamProvenanceUsage     = "Preserve upstream version."
	FlagUpstreamProvenanceDefault   = true

	FlagPatchName      = "patch"
	FlagPatchShorthand = "p"
	FlagPatchUsage     = "Patch diffs from a previous version with its upstream version."
	FlagPatchDefault   = true

	FlagPatchVersionName       = "patch-version"
	FlagPatchVersionShorthand  = "z"
	FlagPatchVersionUsage      = "Version to apply patch from."
	FlagPatchVersionExample001 = "0.0.1"

	FlagDiffSourceName      = "diff-source"
	FlagDiffSourceShorthand = "s"
	FlagDiffSourceUsage     = "Source directory to diff against."
	FlagDiffSourceExample   = "/root/charts/keycloakx/2.1.0"

	FlagPatchFileName      = "patch-file"
	FlagPatchFileShorthand = "f"
	FlagPatchFileUsage     = "Patch file location."
	FlagPatchFileExample   = "/root/charts/vz_charts_patch_keycloakx_0.0.1.patch"

	FlagExampleFormat            = "--%s|-%s %v "
	CommandWithFlagExampleFormat = `%s ` + FlagExampleFormat
)
