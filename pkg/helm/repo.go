// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package helm

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
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

//func LookupChartType(log vzlog.VerrazzanoLogger, repoName, repoURI, chartName, chartVersion string) (installv1beta2.ChartType, error) {
//	indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
//	if err != nil {
//		return installv1beta2.UnclassifiedChartType, err
//	}
//	chartVersions := findChartEntry(indexFile, chartName)
//	for _, version := range chartVersions {
//		if version.Version == chartVersion {
//			moduleType, ok := version.Annotations[ModuleTypeAnnotation]
//			if ok {
//				return installv1beta2.ChartType(moduleType), nil
//			}
//		}
//	}
//	return installv1beta2.UnclassifiedChartType, log.ErrorfThrottledNewErr("Unable to load module type for chart %s-v%s in repo %s", chartName, chartVersion, repoURI)
//}

func ApplyModuleDefinitions(log vzlog.VerrazzanoLogger, client client.Client, chartName, chartVersion, repoURI string) error {
	//indexFile, err := loadAndSortRepoIndexFile(repoName, repoURI)
	//if err != nil {
	//	return err
	//}

	// Find module selectedVersion in Helm repo that matches
	//selectedVersion, err := findSupportingChartVersion(log, indexFile, chartName, platformVersion)
	//if err != nil {
	//	return err
	//}
	//if selectedVersion == nil {
	//	return nil
	//}

	if len(chartName) == 0 {
		return log.ErrorfThrottledNewErr("Chart name can not be empty")
	}
	if len(chartVersion) == 0 {
		return log.ErrorfThrottledNewErr("Chart version can not be empty")
	}

	// Download chart and apply resources in moduleDefs dir
	downloadDir := fmt.Sprintf("%s-%s-*", chartName, chartVersion)
	chartTempDir, err := os.MkdirTemp("", downloadDir)
	if err != nil {
		return err
	}
	// FIXME: uncomment to allow cleanup
	//defer vzos.RemoveTempFiles(log.GetRootZapLogger(), chartTempDir)
	log.Progressf("Pulling chart %s:%s to tempdir %s", chartName, chartVersion, chartTempDir)
	if err := Pull(log, repoURI, chartName, chartVersion, chartTempDir, true); err != nil {
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
