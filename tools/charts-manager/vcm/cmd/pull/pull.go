// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pull

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	vcmhelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/fs"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/helm"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName             = "pull"
	helpShort               = "Pulls an upstream chart/version"
	helpLong                = `The command 'pull' pulls an upstream chart from a given helm repo url and unpacks that to a charts directory.`
	ErrPatchVersionNotFound = "unable to find version to patch, error %v"
	ErrPatchNotGenerated    = "unable to generate patch file, error %v"
	ErrPatchNotApplied      = "unable to apply patch file %s, error %v"
	ErrPatchNotRemoved      = "unable to remove patch file at %v, error %v"
	ErrPatchReview          = "please review patch file at %s and applied changes"
)

func buildExample() string {
	examples := []string{fmt.Sprintf(constants.CommandWithFlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat,
		CommandName, constants.FlagChartName, constants.FlagChartShorthand, constants.FlagChartExampleKeycloak,
		constants.FlagVersionName, constants.FlagPatchVersionShorthand, constants.FlagVersionExample210,
		constants.FlagDirName, constants.FlagDirShorthand, constants.FlagDirExampleLocal,
		constants.FlagRepoName, constants.FlagRepoShorthand, constants.FlagRepoExampleCodecentric)}

	examples = append(examples, "\n")
	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagTargetVersionName, constants.FlagChartShorthand, constants.FlagTargetVersionExample002))

	examples = append(examples, "\n")
	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagUpstreamProvenanceName, constants.FlagUpstreamProvenanceShorthand, constants.FlagUpstreamProvenanceDefault))

	examples = append(examples, "\n")
	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagPatchName, constants.FlagPatchShorthand, constants.FlagPatchDefault))

	examples = append(examples, "\n")
	examples = append(examples, fmt.Sprintf(constants.CommandWithFlagExampleFormat, examples[len(examples)-1],
		constants.FlagPatchVersionName, constants.FlagPatchVersionShorthand, constants.FlagPatchVersionExample001))

	return fmt.Sprintln(examples)
}

// NewCmdPull creates a new instance of pull cmd.
func NewCmdPull(vzHelper helpers.VZHelper, inHfs fs.ChartFileSystem, inHelmConfig helm.HelmConfig) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var hfs fs.ChartFileSystem
		var helmConfig helm.HelmConfig
		if inHfs == nil {
			hfs = fs.HelmChartFileSystem{}
		} else {
			hfs = inHfs
		}

		if inHelmConfig == nil {
			var err error
			helmConfig, err = helm.NewHelmConfig(vzHelper)
			if err != nil {
				return fmt.Errorf("unable to init helm config, error %v", err)
			}
		} else {
			helmConfig = inHelmConfig
		}

		return runCmdPull(cmd, vzHelper, hfs, helmConfig)
	}
	cmd.Example = buildExample()
	cmd.PersistentFlags().StringP(constants.FlagChartName, constants.FlagChartShorthand, "", constants.FlagChartUsage)
	cmd.PersistentFlags().StringP(constants.FlagVersionName, constants.FlagVersionShorthand, "", constants.FlagVersionUsage)
	cmd.PersistentFlags().StringP(constants.FlagDirName, constants.FlagDirShorthand, "", constants.FlagDirUsage)
	cmd.PersistentFlags().StringP(constants.FlagRepoName, constants.FlagRepoShorthand, "", constants.FlagRepoUsage)
	cmd.PersistentFlags().StringP(constants.FlagTargetVersionName, constants.FlagTargetVersionShorthand, "", constants.FlagTargetVersionUsage)
	cmd.PersistentFlags().BoolP(constants.FlagUpstreamProvenanceName, constants.FlagUpstreamProvenanceShorthand, constants.FlagUpstreamProvenanceDefault, constants.FlagUpstreamProvenanceUsage)
	cmd.PersistentFlags().BoolP(constants.FlagPatchName, constants.FlagPatchShorthand, constants.FlagPatchDefault, constants.FlagPatchUsage)
	cmd.PersistentFlags().StringP(constants.FlagPatchVersionName, constants.FlagPatchVersionShorthand, "", constants.FlagPatchVersionUsage)

	return cmd
}

// runCmdPull - run the "vcm pull" command to pull a chart/version from a helm repo and unpack into a chart dierctory with either the same version
// or a target version. Also saves the original chart in a upstream directory and applies any changes that were done to prevopus versions of
// this chart in the same charts directory.
func runCmdPull(cmd *cobra.Command, vzHelper helpers.VZHelper, hfs fs.ChartFileSystem, helmConfig helm.HelmConfig) error {
	chart, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagChartName, constants.FlagChartShorthand)
	if err != nil {
		return err
	}

	version, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagVersionName, constants.FlagVersionShorthand)
	if err != nil {
		return err
	}

	chartsDir, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagDirName, constants.FlagDirShorthand)
	if err != nil {
		return err
	}

	repo, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagRepoName, constants.FlagRepoShorthand)
	if err != nil {
		return err
	}

	targetVersion, err := cmd.PersistentFlags().GetString(constants.FlagTargetVersionName)
	if err != nil {
		return err
	}

	if targetVersion == "" {
		targetVersion = version
	}

	if len(strings.TrimSpace(targetVersion)) == 0 {
		return fmt.Errorf(vcmhelpers.ErrFormatNotEmpty, constants.FlagTargetVersionName)
	}

	saveUpstream, err := cmd.PersistentFlags().GetBool(constants.FlagUpstreamProvenanceName)
	if err != nil {
		return err
	}

	patchDiffs, err := cmd.PersistentFlags().GetBool(constants.FlagPatchName)
	if err != nil {
		return err
	}

	var patchVersion string
	if patchDiffs {
		patchVersion, err = cmd.PersistentFlags().GetString(constants.FlagPatchVersionName)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Adding/Updtaing %s chart repo with url %s..\n", chart, repo)
	repoName, err := helmConfig.AddAndUpdateChartRepo(chart, repo)
	if err != nil {
		return err
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Pulling %s chart version %s to target version %s..\n", chart, version, targetVersion)
	err = helmConfig.DownloadChart(chart, repoName, version, targetVersion, chartsDir)
	if err != nil {
		return err
	}

	err = hfs.RearrangeChartDirectory(chartsDir, chart, targetVersion)
	if err != nil {
		return err
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "Pulled chart %s version %s to target version %s from %s to %s/%s/%s.\n", chart, version, targetVersion, repo, chartsDir, chart, targetVersion)
	if saveUpstream {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Saving upstream chart..\n")
		err = hfs.SaveUpstreamChart(chartsDir, chart, version, targetVersion)
		if err != nil {
			return err
		}

		upstreamChartDir, err := filepath.Abs(fmt.Sprintf("%s/../provenance/%s/upstreams/%s", chartsDir, chart, version))
		if err != nil {
			return err
		}

		fmt.Fprintf(vzHelper.GetOutputStream(), "Upstream chart saved to %s.\n", upstreamChartDir)
		fmt.Fprintf(vzHelper.GetOutputStream(), "Saving chart provenance file..\n")
		chartProvenance, err := helmConfig.GetChartProvenance(chart, repo, version)
		if err != nil {
			return err
		}

		err = hfs.SaveChartProvenance(chartsDir, chartProvenance, chart, targetVersion)
		if err != nil {
			return err
		}

		provenanceFile, err := filepath.Abs(fmt.Sprintf("%s/../provenance/%s/%s.yaml", chartsDir, chart, targetVersion))
		if err != nil {
			return err
		}

		fmt.Fprintf(vzHelper.GetOutputStream(), "Upstream provenance manifest created in %s.\n", provenanceFile)
	}

	if patchDiffs {
		if patchVersion == "" {
			patchVersion, err = hfs.FindChartVersionToPatch(chartsDir, chart, targetVersion)
			if err != nil {
				return fmt.Errorf(ErrPatchVersionNotFound, err)
			}
		}

		if patchVersion != "" {
			patchFile, err := hfs.GeneratePatchFile(chartsDir, chart, patchVersion)
			if err != nil {
				return fmt.Errorf(ErrPatchNotGenerated, err)
			}

			if patchFile == "" {
				fmt.Fprintf(vzHelper.GetOutputStream(), "Nothing to patch from previous version.\n")
				return nil
			}

			rejectsFileGenerated, err := hfs.ApplyPatchFile(chartsDir, vzHelper, chart, targetVersion, patchFile)
			if err != nil {
				return fmt.Errorf(ErrPatchNotApplied, patchFile, err)
			}

			if !rejectsFileGenerated {
				fmt.Fprintf(vzHelper.GetOutputStream(), "Any diffs from version %s has been applied.\n", patchVersion)
				err = os.Remove(patchFile)
				if err != nil {
					return fmt.Errorf(ErrPatchNotRemoved, patchFile, err)
				}
			} else {
				return fmt.Errorf(ErrPatchReview, patchFile)
			}
		}
	}
	return nil
}
