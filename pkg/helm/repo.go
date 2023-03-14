// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package helm

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	platformv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/platform/v1alpha1"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SupportedVersionsAnnotation = "verrazzano.io/supported-versions"
	ModuleTypeAnnotation        = "verrazzano.io/module-type"
)

// TODO: One possible option for these impls is to create our own (internal) Helm plugin and using that with the Helm CLI for applying
//   our own chart conventions, version lookups, etc

func ListChartsInRepo(log vzlog.VerrazzanoLogger, repoName string, repoURL string) error {
	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURL)
	if err != nil {
		return err
	}
	for name, chartVersions := range indexFile.Entries {
		for _, chartVersion := range chartVersions {
			log.Infof("Chart name: %s, version: %v, annotations: %v", name, chartVersion.Version, chartVersion.Metadata.Annotations)
		}
	}
	return nil
}

func LookupChartType(log vzlog.VerrazzanoLogger, repoName, repoURI, chartName, chartVersion string) (platformv1alpha1.ChartType, error) {
	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
	if err != nil {
		return platformv1alpha1.UnclassifiedChartType, err
	}
	chartVersions := findChartEntry(indexFile, chartName)
	for _, version := range chartVersions {
		if version.Version == chartVersion {
			moduleType, ok := version.Annotations[ModuleTypeAnnotation]
			if ok {
				return platformv1alpha1.ChartType(moduleType), nil
			}
		}
	}
	return platformv1alpha1.UnclassifiedChartType, log.ErrorfThrottledNewErr("Unable to load module type for chart %s-v%s in repo %s", chartName, chartVersion, repoURI)
}

func ApplyModuleDefinitions(log vzlog.VerrazzanoLogger, client client.Client, chartName, repoName, repoURI, platformVersion string) error {
	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
	if err != nil {
		return err
	}

	// Find module selectedVersion in Helm repo that matches
	selectedVersion, err := findSupportingChartVersion(log, indexFile, chartName, platformVersion)
	if err != nil {
		return err
	}
	if selectedVersion == nil {
		return nil
	}

	// Download chart and apply resources in moduleDefs dir
	downloadDir := fmt.Sprintf("%s-%s-*", selectedVersion.Name, selectedVersion.Version)
	chartTempDir, err := os.MkdirTemp("", downloadDir)
	if err != nil {
		return err
	}
	// FIXME: uncomment to allow cleanup
	//defer vzos.RemoveTempFiles(log.GetRootZapLogger(), chartTempDir)
	log.Progressf("Pulling chart %s:%s to tempdir %s", selectedVersion.Name, selectedVersion.Version, chartTempDir)
	if err := Pull(log, repoURI, selectedVersion.Name, selectedVersion.Version, chartTempDir, true); err != nil {
		return err
	}
	return ApplyModuleDefsYaml(log, client, fmt.Sprintf("%s/%s", chartTempDir, chartName))
}

// FIXME: This should be with the same set of utils under the VPO, but that or this code would need to be refactored accordingly

// ApplyModuleDefsYaml Applys the set of resources under the "moduleDefs" directory if it exists
func ApplyModuleDefsYaml(log vzlog.VerrazzanoLogger, c client.Client, chartDir string) error {
	// TODO: NewYAMLApplier should probably be enhanced to allow templating if we do this
	path := filepath.Join(chartDir, "/moduleDefs")
	yamlApplier := k8sutil.NewYAMLApplier(c, "")
	log.Oncef("Applying module defs for chart %s at path %s", path)
	return yamlApplier.ApplyD(path)
}

// FindNearestSupportingChartVersion Finds the most recent ChartVersion that meets the platform version specified
func FindNearestSupportingChartVersion(log vzlog.VerrazzanoLogger, chartName, repoName, repoURI, forPlatformVersion string) (string, error) {
	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
	if err != nil {
		return "", err
	}
	version, err := findSupportingChartVersion(log, indexFile, chartName, forPlatformVersion)
	if err != nil {
		return "", err
	}
	return version.Version, nil
}

// findSupportingChartVersion Finds the most recent ChartVersion that
func findSupportingChartVersion(log vzlog.VerrazzanoLogger, indexFile *repo.IndexFile, chartName string, forPlatformVersion string) (*repo.ChartVersion, error) {
	// The indexFile is already sorted in descending order for each chart
	chartVersions := findChartEntry(indexFile, chartName)
	for _, version := range chartVersions {
		supportedVzVersionsConstraint, ok := version.Annotations[SupportedVersionsAnnotation]
		if ok {
			matches, err := semver.MatchesConstraint(forPlatformVersion, supportedVzVersionsConstraint)
			if err != nil {
				return nil, err
			}
			if matches {
				return version, nil
			}
		}
	}
	log.Infof("No compatible version for chart %s found in repo for platform version %s", chartName, forPlatformVersion)
	return nil, nil
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

	// TODO: is this cached?
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
