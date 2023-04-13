// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"strings"
)

const (
	SupportedVersionsAnnotation = "verrazzano.io/supported-versions"
	ModuleTypeAnnotation        = "verrazzano.io/module-type"
)

type HelmReleaseOpts struct {
	RepoURL      string
	ReleaseName  string
	Namespace    string
	ChartPath    string
	ChartVersion string
	Overrides    []helm.HelmOverrides

	Username string
	Password string
}

func UpgradeRelease(log vzlog.VerrazzanoLogger, releaseOpts *HelmReleaseOpts, wait bool, dryRun bool) (*release.Release, error) {
	log.Infof("Upgrading release %s in namespace %s, chart %s, version %s, repoURL %s", releaseOpts.ReleaseName,
		releaseOpts.Namespace, releaseOpts.ChartPath, releaseOpts.ChartVersion, releaseOpts.RepoURL)
	settings := cli.New()
	settings.SetNamespace(releaseOpts.Namespace)

	chartOptions := action.ChartPathOptions{
		RepoURL:  releaseOpts.RepoURL,
		Version:  releaseOpts.ChartVersion,
		Password: releaseOpts.Username,
		Username: releaseOpts.Password,
	}

	chartPath, err := chartOptions.LocateChart(releaseOpts.ChartPath, settings)
	if err != nil {
		return nil, err
	}

	return helm.Upgrade(log, releaseOpts.ReleaseName, releaseOpts.Namespace, chartPath, wait, dryRun, releaseOpts.Overrides)
}

// GetReleaseChartVersion extracts the chart version from a deployed helm release
func GetReleaseChartVersion(releaseName string, namespace string) (string, error) {
	releases, err := helm.GetReleases(namespace)
	if err != nil {
		if err.Error() == helm.ChartNotFound {
			return helm.ChartNotFound, nil
		}
		return "", err
	}

	var version string
	for _, info := range releases {
		release := info.Name
		if release == releaseName {
			version = info.Chart.Metadata.Version
			break
		}
	}
	return strings.TrimSpace(version), nil
}

// FindLatestChartVersion Finds the most recent ChartVersion
func FindLatestChartVersion(log vzlog.VerrazzanoLogger, chartName, repoName, repoURI string) (string, error) {
	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
	if err != nil {
		return "", err
	}
	version, err := findMostRecentChartVersion(log, indexFile, chartName)
	if err != nil {
		return "", err
	}
	return version.Version, nil
}

// findMostRecentChartVersion Finds the most recent ChartVersion that
func findMostRecentChartVersion(log vzlog.VerrazzanoLogger, indexFile *repo.IndexFile, chartName string) (*repo.ChartVersion, error) {
	// The indexFile is already sorted in descending order for each chart
	chartVersions := findChartEntry(indexFile, chartName)
	if len(chartVersions) == 0 {
		return nil, fmt.Errorf("no entries found for chart %s in repo", chartName)
	}
	return chartVersions[0], nil
}

func findChartEntry(index *repo.IndexFile, chartName string) repo.ChartVersions {
	var selectedVersion repo.ChartVersions
	for name, chartVersions := range index.Entries {
		if name == chartName {
			selectedVersion = chartVersions
		}
	}
	return selectedVersion
}

func loadAndSortRepoIndexFile(repoName string, repoURL string) (*repo.IndexFile, error) {
	// NOTES:
	// - we'll need to allow defining credentials etc in the source lists for protected repos
	// - also we'll likely need better scaffolding around local repo management
	cfg := &repo.Entry{
		Name: repoName,
		URL:  repoURL,
	}
	chartRepository, err := repo.NewChartRepository(cfg, getter.All(cli.New()))
	if err != nil {
		return nil, err
	}
	indexFilePath, err := chartRepository.DownloadIndexFile()
	if err != nil {
		return nil, err
	}
	indexFile, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		return nil, err
	}
	indexFile.SortEntries()
	return indexFile, nil
}
