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

func ListCharts(log vzlog.VerrazzanoLogger, repoName string, repoURL string) error {
	indexFile, err := loadRepoIndexFile(repoName, repoURL)
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

func LookupChartType(log vzlog.VerrazzanoLogger, repoName, repoURL, chartName string) platformv1alpha1.ChartType {
	// TODO: Hard-coded for now, implement lookup via Chart annotations in repo index
	switch chartName {
	case "mysql-operator":
		return platformv1alpha1.OperatorChartType
	default:
		return platformv1alpha1.ModuleChartType
	}
	return platformv1alpha1.UnclassifiedChartType
}

func LoadModuleDefinitions(log vzlog.VerrazzanoLogger, client client.Client, chartName, repoName, repoURI, platformVersion string) error {
	indexFile, err := loadRepoIndexFile(repoName, repoURI)
	if err != nil {
		return err
	}
	indexFile.SortEntries()

	// Find module selectedVersion in Helm repo that matches
	selectedVersion, err := findSupportingChartVersion(indexFile, chartName)
	if err != nil {
		return err
	}
	if selectedVersion == nil {
		return nil
	}

	// Download chart and apply resources in moduleDefs dir
	downloadDir := fmt.Sprintf("%s/%s-%s", "/tmp/", selectedVersion.Name, selectedVersion.Version)
	if err := os.Mkdir(downloadDir, 0777); err != nil {
		return err
	}
	if err := Pull(log, repoURI, selectedVersion.Name, selectedVersion.Version, downloadDir, true); err != nil {
		return err
	}
	return ApplyModuleDefsYaml(log, client, fmt.Sprintf("%s/%s", downloadDir, chartName))
}

func ApplyModuleDefsYaml(log vzlog.VerrazzanoLogger, c client.Client, chartDir string) error {
	path := filepath.Join(chartDir, "/moduleDefs")
	yamlApplier := k8sutil.NewYAMLApplier(c, "")
	log.Oncef("Applying module defs for chart %s at path %s", path)
	return yamlApplier.ApplyD(path)
}

func findSupportingChartVersion(indexFile *repo.IndexFile, chartName string) (*repo.ChartVersion, error) {
	chartVersions := findChartEntry(indexFile, chartName)
	for _, version := range chartVersions {
		supportedVersions, ok := version.Annotations[SupportedVersionsAnnotation]
		if ok {
			matches, err := semver.MatchesConstraint(version.Version, supportedVersions)
			if err != nil {
				return nil, err
			}
			if matches {
				return version, nil
			}
		}
	}
	return nil, nil
}

func findChartEntry(index *repo.IndexFile, chartName string) repo.ChartVersions {
	var chartVersions repo.ChartVersions
	for name, chartVersions := range index.Entries {
		if name == chartName {
			chartVersions = chartVersions
		}
	}
	return chartVersions
}

func loadRepoIndexFile(repoName string, repoURL string) (*repo.IndexFile, error) {
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
	return indexFile, nil
}
