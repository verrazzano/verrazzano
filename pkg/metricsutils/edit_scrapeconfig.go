// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsutils

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"sigs.k8s.io/yaml"
)

// EditScrapeJob edits a scrape config and adds or replaces the specified job with the new scrape
// config for that job.
func EditScrapeJob(scrapeConfigs *gabs.Container, editScrapeJobName string, newScrapeConfig *gabs.Container) (*gabs.Container, error) {
	scrapeJobIndex := FindScrapeJob(scrapeConfigs, editScrapeJobName)
	found := scrapeJobIndex >= 0
	// found an existing scrape config, either remove it or replace it
	if found {
		if newScrapeConfig == nil || newScrapeConfig.Data() == nil {
			scrapeConfigs.ArrayRemove(scrapeJobIndex)
		} else {
			scrapeConfigs.SetIndex(newScrapeConfig.Data(), scrapeJobIndex)
		}
	}

	if !found && newScrapeConfig != nil {
		// if we didn't find an existing scrape config and we are not removing it, append it to the existing scrape config
		scrapeConfigs.ArrayAppend(newScrapeConfig.Data())
	}

	return scrapeConfigs, nil
}

// EditScrapeJobInPrometheusConfig is similar to EditScrapeJob, but it takes a prometheus config
// as first parameter i.e. a container containing a "scrape_configs" key (or other key specified by the scrapeJobsKey parameter)
// This may appear to duplicate some of the code in EditScrapeJob, but it has to be in a separate
// implementation because of the way gabs.Container behaves when editing arrays embedded in a parent
// object versus directly editing a top level array element. In this case, the underlying parent
// object does not get updated correctly.
func EditScrapeJobInPrometheusConfig(promConfig *gabs.Container, scrapeJobsKey string, editScrapeJobName string, newScrapeConfig *gabs.Container) error {
	scrapeConfigs := promConfig.Search(scrapeJobsKey)
	scrapeJobIndex := FindScrapeJob(scrapeConfigs, editScrapeJobName)
	// _, err := EditScrapeJob(scrapeConfigs, editScrapeJobName, newScrapeConfig)
	// return err
	found := scrapeJobIndex >= 0
	// found an existing scrape config, either remove it or replace it
	if found {
		if newScrapeConfig == nil || newScrapeConfig.Data() == nil {
			err := promConfig.ArrayRemove(scrapeJobIndex, scrapeJobsKey)
			return err
		}
		_, err := scrapeConfigs.SetIndex(newScrapeConfig.Data(), scrapeJobIndex)
		return err
	}

	if !found && newScrapeConfig != nil {
		// if we didn't find an existing scrape config and we are not removing it, append it to the existing scrape config
		return promConfig.ArrayAppend(newScrapeConfig.Data(), scrapeJobsKey)
	}

	return nil
}

// FindScrapeJob returns the index of the given job name in the scrapeConfigs list, -1 if not found.
func FindScrapeJob(scrapeConfigs *gabs.Container, jobNameToFind string) int {
	for index, scrapeConfig := range scrapeConfigs.Children() {
		scrapeJobName := scrapeConfig.Search(constants.PrometheusJobNameKey).Data()
		if jobNameToFind == scrapeJobName {
			return index
		}
	}
	return -1
}

// ParseScrapeConfig returns an editable representation of the prometheus scrape configuration
func ParseScrapeConfig(scrapeConfigStr string) (*gabs.Container, error) {
	scrapeConfigJSON, _ := yaml.YAMLToJSON([]byte(scrapeConfigStr))
	newScrapeConfig, err := gabs.ParseJSON(scrapeConfigJSON)
	if err != nil {
		return nil, err
	}
	return newScrapeConfig, nil
}
