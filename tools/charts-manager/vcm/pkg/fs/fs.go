// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fs

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"gopkg.in/yaml.v3"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// ChartFileSystem represents all the file operations that are performed for helm charts.
type ChartFileSystem interface {
	RearrangeChartDirectory(string, string, string) error
	SaveUpstreamChart(string, string, string, string) error
	SaveChartProvenance(string, *helm.ChartProvenance, string, string) error
	GeneratePatchFile(string, string, string) (string, error)
	GeneratePatchWithSourceDir(string, string, string, string) (string, error)
	FindChartVersionToPatch(string, string, string) (string, error)
	ApplyPatchFile(string, helpers.VZHelper, string, string, string) (bool, error)
}

// HelmChartFileSystem is the default implementation of ChartFileSystem.
type HelmChartFileSystem struct{}

// RearrangeChartDirectory moves the downloaded chart directory one level up in the supplied chartsDir so that the structure is
// <chartsDir>/<chart>/<version>/<all chart files>.
func (hfs HelmChartFileSystem) RearrangeChartDirectory(chartsDir string, chart string, targetVersion string) error {
	pulledChartDir := fmt.Sprintf("%s/%s/%s/%s/", chartsDir, chart, targetVersion, chart)
	targetChartDir := fmt.Sprintf("%s/%s/%s", chartsDir, chart, targetVersion)
	cmd := exec.Command("cp", "-R", pulledChartDir, targetChartDir)
	_, _, err := runner.Run(cmd)
	if err != nil {
		return err
	}

	err = os.RemoveAll(pulledChartDir)
	if err != nil {
		return err
	}
	return nil
}

// SaveUpstreamChart copies the original chart to <chartsDir>/../provenance/<chart>/upstreams/<version> so that upstream is
// persisted.
func (hfs HelmChartFileSystem) SaveUpstreamChart(chartsDir string, chart string, version string, targetVersion string) error {
	upstreamDir := fmt.Sprintf("%s/../provenance/%s/upstreams/%s", chartsDir, chart, version)
	err := os.RemoveAll(upstreamDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(upstreamDir, 0755)
	if err != nil {
		return err
	}

	chartDir := fmt.Sprintf("%s/%s/%s/", chartsDir, chart, targetVersion)
	cmd := exec.Command("cp", "-R", chartDir, upstreamDir)
	_, _, err = runner.Run(cmd)
	return err
}

// SaveChartProvenance serializes the chartProvenance generated for a chart to <chartsDir>/../provenance/<chart>/<targetVersion>.yaml.
func (hfs HelmChartFileSystem) SaveChartProvenance(chartsDir string, chartProvenance *helm.ChartProvenance, chart string, targetVersion string) error {
	provenanceDir := fmt.Sprintf("%s/../provenance/%s", chartsDir, chart)
	err := os.MkdirAll(provenanceDir, 0755)
	if err != nil {
		return err
	}

	provenanceFile := fmt.Sprintf("%s/%s.yaml", provenanceDir, targetVersion)
	out, err := yaml.Marshal(chartProvenance)
	if err != nil {
		return err
	}

	return os.WriteFile(provenanceFile, out, 0600)
}

// GeneratePatchFile diffs between the chart/version present in a chart directory against its upstream chart and generates
// a patch file.
func (hfs HelmChartFileSystem) GeneratePatchFile(chartsDir string, chart string, version string) (string, error) {
	provenanceFile := fmt.Sprintf("%s/../provenance/%s/%s.yaml", chartsDir, chart, version)
	if _, err := os.Stat(provenanceFile); err != nil {
		return "", fmt.Errorf("provenance file %s not found, error %v", provenanceFile, err)
	}

	in, err := os.ReadFile(provenanceFile)
	if err != nil {
		return "", fmt.Errorf("unable to read provenance file %s, error %v", provenanceFile, err)
	}

	chartProvenance := helm.ChartProvenance{}
	err = yaml.Unmarshal(in, &chartProvenance)
	if err != nil {
		return "", fmt.Errorf("unable to parse provenance file %s, error %v", provenanceFile, err)
	}

	return hfs.GeneratePatchWithSourceDir(chartsDir, chart, version, fmt.Sprintf("%s/../provenance/%s/%s", chartsDir, chart, chartProvenance.UpstreamChartLocalPath))

}

// GeneratePatchWithSourceDir diffs a chart against a given directory and generates a patch file.
func (hfs HelmChartFileSystem) GeneratePatchWithSourceDir(chartsDir string, chart string, version string, sourceDir string) (string, error) {
	chartDir := fmt.Sprintf("%s/%s/%s", chartsDir, chart, version)
	if _, err := os.Stat(chartDir); err != nil {
		return "", fmt.Errorf("chart directory %s not found, error %v", chartDir, err)
	}

	sourceChartDirectory, err := filepath.Abs(sourceDir)
	if err != nil {
		return "", fmt.Errorf("unable to find absolute path to upstream/source chart directory at %s, error %v", sourceChartDirectory, err)
	}

	if _, err := os.Stat(sourceChartDirectory); err != nil {
		return "", fmt.Errorf("upstream/source chart directory %s not found, error %v", sourceChartDirectory, err)
	}

	patchFilePathAbsolute, err := filepath.Abs(fmt.Sprintf("%s/../vz_charts_patch_%s_%s.patch", chartsDir, chart, version))
	if err != nil {
		return "", fmt.Errorf("unable to find absolute path for patch file")
	}

	patchFile, err := os.Create(patchFilePathAbsolute)
	if err != nil {
		return "", fmt.Errorf("unable to create empty patch file")
	}

	cmd := exec.Command("diff", "-Naurw", sourceChartDirectory, chartDir)
	cmd.Stdout = patchFile
	err = cmd.Run()
	if err != nil {
		// diff returning exit status 1 even when file diff is completed and no underlying error.
		// error out only when message is different
		if err.Error() != "exit status 1" {
			return "", fmt.Errorf("error running command %s, error %v", cmd.String(), err)
		}
	}

	patchFileStats, err := os.Stat(patchFile.Name())
	if err != nil {
		return "", fmt.Errorf("unable to stat patch file at %v, error %v", patchFile.Name(), err)
	}

	if patchFileStats.Size() == 0 {
		err := os.Remove(patchFile.Name())
		if err != nil {
			return "", fmt.Errorf("unable to remove empty patch file at %v, error %v", patchFile.Name(), err)
		}

		return "", nil
	}

	return patchFile.Name(), nil
}

// FindChartVersionToPatch looks up the last higheset version present in the charts directory against a given chart version.
func (hfs HelmChartFileSystem) FindChartVersionToPatch(chartsDir string, chart string, version string) (string, error) {
	chartDirParent := fmt.Sprintf("%s/%s", chartsDir, chart)
	entries, err := os.ReadDir(chartDirParent)
	if err != nil {
		return "", fmt.Errorf("unable to read chart dierctory %s, error %v", chartDirParent, err)
	}

	currentChartVersion, err := semver.NewSemVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid chart version %s, error %v", version, err)
	}

	var versions []*semver.SemVersion
	for _, entry := range entries {
		if entry.IsDir() {
			chartVersion, err := semver.NewSemVersion(entry.Name())
			if err != nil {
				return "", fmt.Errorf("invalid chart version %s, error %v", chartVersion.ToString(), err)
			}

			if chartVersion.IsLessThan(currentChartVersion) {
				versions = append(versions, chartVersion)
			}
		}
	}

	if len(versions) == 0 {
		return "", nil
	}

	highestVersion := versions[0]
	for _, version := range versions {
		if version.IsGreatherThan(highestVersion) {
			highestVersion = version
		}
	}
	return highestVersion.ToString(), nil
}

// ApplyPatchFile patches a given patch file on a chart.
func (hfs HelmChartFileSystem) ApplyPatchFile(chartsDir string, vzHelper helpers.VZHelper, chart string, version string, patchFile string) (bool, error) {
	chartDir := fmt.Sprintf("%s/%s/%s/", chartsDir, chart, version)
	if _, err := os.Stat(chartDir); err != nil {
		return false, fmt.Errorf("chart directory %s not found, error %v", chartDir, err)
	}

	if _, err := os.Stat(patchFile); err != nil {
		return false, fmt.Errorf("patch file %s not found, error %v", patchFile, err)
	}

	rejectsFilePathAbsolute, err := filepath.Abs(fmt.Sprintf("%s/../vz_charts_patch_%s_%s_rejects.rejects", chartsDir, chart, version))
	if err != nil {
		return false, fmt.Errorf("unable to find absolute path for rejects file")
	}

	_, err = os.Create(rejectsFilePathAbsolute)
	if err != nil {
		return false, fmt.Errorf("unable to create empty rejects file")
	}

	in, err := os.OpenFile(patchFile, io.SeekStart, os.ModePerm)
	if err != nil {
		return false, fmt.Errorf("unable to read patch file")
	}

	skipLevels := fmt.Sprintf("-p%v", fmt.Sprint(strings.Count(chartDir, string(os.PathSeparator))))
	cmd := exec.Command("patch", "--no-backup-if-mismatch", skipLevels, "-r", rejectsFilePathAbsolute, "--directory", chartDir)
	cmd.Stdin = in
	out, cmderr := cmd.CombinedOutput()
	if cmderr != nil && cmderr.Error() != "exit status 1" {
		return false, fmt.Errorf("error running command %s, error %v", cmd.String(), err)
	}

	rejectsFileStats, err := os.Stat(rejectsFilePathAbsolute)
	if err != nil {
		return false, fmt.Errorf("unable to stat reject file at %v, error %v", rejectsFilePathAbsolute, err)
	}

	if rejectsFileStats.Size() == 0 {
		err := os.Remove(rejectsFilePathAbsolute)
		if err != nil {
			return false, fmt.Errorf("unable to remove empty rejects file at %v, error %v", rejectsFilePathAbsolute, err)
		}

		rejectsFilePathAbsolute = ""
	}

	if cmderr != nil && rejectsFilePathAbsolute == "" {
		return false, fmt.Errorf("error running command %s, error %v", cmd.String(), err)
	}

	if len(out) > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), "%s", string(out))
	}

	if rejectsFilePathAbsolute != "" {
		rejects, err := os.ReadFile(rejectsFilePathAbsolute)
		if err != nil {
			return true, fmt.Errorf("unable to read rejects file at %s, error %v", rejectsFilePathAbsolute, err)
		}

		fmt.Fprintf(vzHelper.GetOutputStream(), "%s", string(rejects))
		return true, nil
	}

	return false, nil
}
