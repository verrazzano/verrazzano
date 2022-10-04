// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsutils

import (
	"io/ioutil"
	"testing"

	"github.com/Jeffail/gabs/v2"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/constants"
)

const newJobName = "newjob"
const existingJobName = "prometheus"
const newScrapeJob = constants.PrometheusJobNameKey + `: ` + newJobName + `
kubernetes_sd_configs:
- role: endpoints
relabel_configs:
- action: keep
  regex: node-exporter
  source_labels:
  - __meta_kubernetes_endpoints_name
scrape_interval: 20s
scrape_timeout: 15s
`

const replaceExistingScrapeJob = constants.PrometheusJobNameKey + `: ` + existingJobName + `
scrape_interval: 20s
scrape_timeout: 15s
static_configs:
- targets:
  - localhost:9191
`

// TestEditScrapeJob tests the editing of a list of scrape configs (in the format expected in
// a Prometheus config map's "scrape_configs" section or additionalScrapeConfigs secret)
// GIVEN an updated scrape config job and its name
// WHEN the function is called
// THEN the scrape config job should be either added to the scrape configs list, or updated if a
//
//	job with that name already exists.
func TestEditScrapeJob(t *testing.T) {
	tests := []struct {
		name           string
		editJobName    string
		editConfigData string
		expectAdd      bool // true if new job should be added, false if it's an existing job
		expectRemove   bool // true if existing job should be removed
	}{
		{"add new job", newJobName, newScrapeJob, true, false},
		{"edit existing job", existingJobName, replaceExistingScrapeJob, false, false},
		{"remove job", existingJobName, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scrapejobsBytes, err := ioutil.ReadFile("testdata/scrapejobs.yaml")
			if err != nil {
				t.Errorf("Failed to read test scrape jobs file: %v", err)
			}

			newScrapeConfig, err := ParseScrapeConfig(tt.editConfigData)
			if err != nil {
				t.Errorf("Failed to parse the scrape config for job %s: %v", tt.editJobName, err)
			}

			scrapejobs, err := ParseScrapeConfig(string(scrapejobsBytes))
			if err != nil {
				t.Errorf("Failed to parse the scrape jobs from test data: %v", err)
			}
			origNumScrapeJobs := len(scrapejobs.Children())
			updatedScrapeJobs, err := EditScrapeJob(scrapejobs, tt.editJobName, newScrapeConfig)
			assert.Nil(t, err)
			assertScrapeJobs(t, updatedScrapeJobs, newScrapeConfig, origNumScrapeJobs, tt.editJobName, tt.expectAdd, tt.expectRemove)
		})
	}
}

// TestEditScrapeJobInPrometheusConfig tests the editing of a full Prometheus configuration (in the format expected in
// a Prometheus config map's "prometheus.yaml" field)
// GIVEN an updated scrape config job and its name
// WHEN the function is called
// THEN the scrape config job should be either added to the scrape configs list, or updated if a
//
//	job with that name already exists.
func TestEditScrapeJobInPrometheusConfig(t *testing.T) {
	tests := []struct {
		name           string
		editJobName    string
		editConfigData string
		expectAdd      bool // true if new job should be added, false if it's an existing job
		expectRemove   bool // true if existing job should be removed
	}{
		{"add new job", newJobName, newScrapeJob, true, false},
		{"edit existing job", existingJobName, replaceExistingScrapeJob, false, false},
		{"remove job", existingJobName, "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			promConfigBytes, err := ioutil.ReadFile("testdata/fullPromConfig.yaml")
			if err != nil {
				t.Errorf("Failed to read test scrape jobs file: %v", err)
			}

			newScrapeConfig, err := ParseScrapeConfig(tt.editConfigData)
			if err != nil {
				t.Errorf("Failed to parse the scrape config for job %s: %v", tt.editJobName, err)
			}

			promConfig, err := ParseScrapeConfig(string(promConfigBytes))
			if err != nil {
				t.Errorf("Failed to parse the test prometheus config data: %v", err)
			}
			scrapejobs := promConfig.Search("scrape_configs")
			origNumScrapeJobs := len(scrapejobs.Children())
			err = EditScrapeJobInPrometheusConfig(promConfig, "scrape_configs", tt.editJobName, newScrapeConfig)
			assert.Nil(t, err)
			updatedScrapeJobs := promConfig.Search("scrape_configs")
			assertScrapeJobs(t, updatedScrapeJobs, newScrapeConfig, origNumScrapeJobs, tt.editJobName, tt.expectAdd, tt.expectRemove)
		})
	}
}

func assertScrapeJobs(t *testing.T, updatedScrapeJobs *gabs.Container, newScrapeConfig *gabs.Container, origNumScrapeJobs int, jobName string, expectAdd bool, expectRemove bool) {
	foundJobIndex := FindScrapeJob(updatedScrapeJobs, jobName)
	if expectAdd {
		// should have been added as a new job
		assert.Equal(t, origNumScrapeJobs+1, len(updatedScrapeJobs.Children()))
	} else if expectRemove {
		// should have been removed
		assert.Equal(t, origNumScrapeJobs-1, len(updatedScrapeJobs.Children()))
		assert.Less(t, foundJobIndex, 0)
		// nothing more to assert for remove case
		return
	} else {
		// should have edited an existing job
		assert.Equal(t, origNumScrapeJobs, len(updatedScrapeJobs.Children()))
	}
	// for cases other than removal, scrape config job should exist and be equal to the updated value
	assert.GreaterOrEqual(t, foundJobIndex, 0)
	assert.Equal(t, newScrapeConfig, updatedScrapeJobs.Children()[foundJobIndex])

}
