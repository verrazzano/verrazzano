// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package constants

const FlagChartName = "chart"
const FlagChartShorthand = "c"
const FlagChartUsage = "Name of the chart."
const FlagChartExampleKeycloak = "keycloakx"

const FlagVersionName = "version"
const FlagVersionShorthand = "v"
const FlagVersionUsage = "Upstream version of the chart."
const FlagVersionExample210 = "2.1.0"

const FlagRepoName = "repo"
const FlagRepoShorthand = "r"
const FlagRepoUsage = "URL of the helm repo."
const FlagRepoExampleCodecentric = "https://codecentric.github.io/helm-charts"

const FlagDirName = "charts-dir"
const FlagDirShorthand = "d"
const FlagDirUsage = "Location of charts directory."
const FlagDirExampleLocal = "./charts"

const FlagTargetVersionName = "target-version"
const FlagTargetVersionShorthand = "t"
const FlagTargetVersionUsage = "Target downstream version of the chart."
const FlagTargetVersionExample002 = "0.0.2"

const FlagUpstreamProvenanceName = "upstream-provenance"
const FlagUpstreamProvenanceShorthand = "u"
const FlagUpstreamProvenanceUsage = "Preserve upstream version."
const FlagUpstreamProvenanceDefault = true

const FlagPatchName = "patch"
const FlagPatchShorthand = "p"
const FlagPatchUsage = "Patch diffs from a previous version with its upstream version."
const FlagPatchDefault = true

const FlagPatchVersionName = "patch-version"
const FlagPatchVersionShorthand = "z"
const FlagPatchVersionUsage = "Version to apply patch from."
const FlagPatchVersionExample001 = "0.0.1"

const FlagDiffSourceName = "diff-source"
const FlagDiffSourceShorthand = "s"
const FlagDiffSourceUsage = "Source directory to diff against."
const FlagDiffSourceExample = "/root/charts/keycloakx/2.1.0"

const FlagPatchFileName = "patch-file"
const FlagPatchFileShorthand = "f"
const FlagPatchFileUsage = "Patch file location."
const FlagPatchFileExample = "/root/charts/vz_charts_patch_keycloakx_0.0.1.patch"

const FlagExampleFormat = "--%s|-%s %v "
const CommandWithFlagExampleFormat = `%s ` + FlagExampleFormat

const HelmCachePath = "/tmp/.helmcache"
const HelmRepoPath = "/tmp/.helmrepo"
