package scrapeconfigutils

import (
	"github.com/Jeffail/gabs/v2"
	"sigs.k8s.io/yaml"
)

const jobNameKey = "job_name"

// EditScrapeJob edits a scrape config that and adds or replaces the specified job with the new scrape
// config for that job.
func EditScrapeJob(scrapeConfigs *gabs.Container, editScrapeJobName string, newScrapeConfig *gabs.Container) (*gabs.Container, error) {
	scrapeJobIndex := findScrapeJob(scrapeConfigs, editScrapeJobName)
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

// findScrapeJob returns the index of the given job name in the scrapeConfigs list, -1 if not found.
func findScrapeJob(scrapeConfigs *gabs.Container, jobNameToFind string) int {
	for index, scrapeConfig := range scrapeConfigs.Children() {
		scrapeJobName := scrapeConfig.Search(jobNameKey).Data()
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
