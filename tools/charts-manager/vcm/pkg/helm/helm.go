// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"os"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	helmrepo "helm.sh/helm/v3/pkg/repo"
)

// HelmConfig represents any helm and related operations done for a helm chart.
type HelmConfig interface {
	AddAndUpdateChartRepo(string, string) (string, error)
	DownloadChart(string, string, string, string, string) error
	GetChartProvenance(string, string, string) (*ChartProvenance, error)
}

// VerrazzanoHelmConfig is default implementation of HelmConfig.
type VerrazzanoHelmConfig struct {
	settings     *cli.EnvSettings
	helmRepoFile *helmrepo.File
	vzHelper     helpers.VZHelper
}

// ChartProvenance represents the upstream provenance information about a chart version by reading/creating
// the helm repository config and cache.
type ChartProvenance struct {
	// UpstreamVersion is version of the upstream chart.
	UpstreamVersion string `yaml:"upstreamVersion"`
	// UpstreamChartLocalPath contains relative path to upstream chart directory.
	UpstreamChartLocalPath string `yaml:"upstreamChartLocalPath"`
	// UpstreamIndexEntry is the index.yaml entry for the upstream chart.
	UpstreamIndexEntry *helmrepo.ChartVersion `yaml:"upstreamIndexEntry"`
}

// NewHelmConfig initializes the default HelmConfig instance.
func NewHelmConfig(vzHelper helpers.VZHelper) (*VerrazzanoHelmConfig, error) {
	helmConfig := &VerrazzanoHelmConfig{vzHelper: vzHelper}
	helmConfig.settings = cli.New()
	if helmConfig.settings.RepositoryConfig == "" {
		helmConfig.settings.RepositoryConfig = "/tmp/vz_helm_repo.yaml"
	}

	if helmConfig.settings.RepositoryCache == "" {
		helmConfig.settings.RepositoryCache = "/tmp/vz_helm_repo_cache"
	}

	err := os.MkdirAll(helmConfig.settings.RepositoryCache, 0755)
	if err != nil {
		return nil, err
	}

	if _, err = os.Stat(helmConfig.settings.RepositoryConfig); err != nil {
		err = helmrepo.NewFile().WriteFile(helmConfig.settings.RepositoryConfig, 0o644)
		if err != nil {
			return nil, err
		}
	}

	helmConfig.helmRepoFile, err = helmrepo.LoadFile(helmConfig.settings.RepositoryConfig)
	if err != nil {
		return nil, err
	}
	return helmConfig, nil
}

// AddAndUpdateChartRepo creates/updates the local helm repo for the given repoURL and
// downloads the index file.
func (h VerrazzanoHelmConfig) AddAndUpdateChartRepo(chart string, repoURL string) (string, error) {
	repoEntry, err := h.getRepoEntry(repoURL)
	if err != nil {
		return "", err
	}

	if repoEntry == nil {
		repoEntry = &helmrepo.Entry{
			Name: fmt.Sprintf("%s-provider", chart),
			URL:  repoURL,
		}
		fmt.Fprintf(h.vzHelper.GetOutputStream(), "Adding helm repo %s.\n", repoEntry.Name)
	} else {
		fmt.Fprintf(h.vzHelper.GetOutputStream(), "Using helm repo %s\n", repoEntry.Name)
	}

	chartRepo, err := helmrepo.NewChartRepository(repoEntry, getter.All(h.settings))
	if err != nil {
		return "", err
	}

	_, err = chartRepo.DownloadIndexFile()
	if err != nil {
		return "", err
	}

	h.helmRepoFile.Update(repoEntry)
	return repoEntry.Name, h.helmRepoFile.WriteFile(h.settings.RepositoryConfig, 0o644)
}

// DownloadChart pulls a chart from the remote helm repo and untars into the chart ditrectory.
func (h VerrazzanoHelmConfig) DownloadChart(chart string, repo string, version string, targetVersion string, chartDir string) error {
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(h.settings.Debug),
		registry.ClientOptCredentialsFile(h.settings.RegistryConfig),
	)
	if err != nil {
		return err
	}

	config := &action.Configuration{
		RegistryClient: registryClient,
	}

	pull := action.NewPullWithOpts(action.WithConfig(config))
	pull.Untar = true
	pull.UntarDir = fmt.Sprintf("%s/%s/%s", chartDir, chart, targetVersion)
	pull.Settings = cli.New()
	pull.Version = version
	err = os.RemoveAll(pull.UntarDir)
	if err != nil {
		return err
	}

	out, err := pull.Run(fmt.Sprintf("%s/%s", repo, chart))
	if out != "" {
		fmt.Fprintln(h.vzHelper.GetOutputStream(), out)
	}

	return err
}

// GetChartProvenance creates the provenance data for a chart version against its upstream.
func (h VerrazzanoHelmConfig) GetChartProvenance(chart string, repo string, version string) (*ChartProvenance, error) {
	repoEntry, err := h.getRepoEntry(repo)
	if err != nil {
		return nil, err
	}

	indexPath := fmt.Sprintf("%s/%s-index.yaml", h.settings.RepositoryCache, repoEntry.Name)
	if _, err = os.Stat(indexPath); err != nil {
		return nil, err
	}

	indexFile, err := helmrepo.LoadIndexFile(indexPath)
	if err != nil {
		return nil, err
	}

	chartVersion, err := indexFile.Get(chart, version)
	if err != nil {
		return nil, err
	}

	return &ChartProvenance{
		UpstreamVersion:        version,
		UpstreamChartLocalPath: fmt.Sprintf("upstreams/%s", version),
		UpstreamIndexEntry:     chartVersion,
	}, nil
}

func (h VerrazzanoHelmConfig) getRepoEntry(repoURL string) (*helmrepo.Entry, error) {
	for _, repoEntry := range h.helmRepoFile.Repositories {
		if repoEntry.URL == repoURL {
			return repoEntry, nil
		}
	}

	return nil, fmt.Errorf("could not find repo entry")
}
