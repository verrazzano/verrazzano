// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package helm

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func ListCharts(log vzlog.VerrazzanoLogger, repoName string, repoURL string) error {

	// NOTES:
	// - we'll need to allow defining credentials etc in the source lists for protected repos
	cfg := &repo.Entry{
		Name: repoName,
		URL:  repoURL,
	}
	chartRepository, err := repo.NewChartRepository(cfg, getter.All(cli.New()))
	if err != nil {
		return err
	}
	indexFilePath, err := chartRepository.DownloadIndexFile()
	if err != nil {
		return err
	}
	indexFile, err := repo.LoadIndexFile(indexFilePath)
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
