package scrapeconfigutils

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

const NEW_JOB_NAME = "newjob"
const EXISTING_JOB_NAME = "prometheus"
const NEW_SCRAPE_JOB = `
job_name: ` + NEW_JOB_NAME + `
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

const REPLACE_EXISTING_SCRAPE_JOB = `
job_name: ` + EXISTING_JOB_NAME + `
scrape_interval: 20s
scrape_timeout: 15s
static_configs:
- targets:
  - localhost:9191
`

// TestEditScrapeJob tests the editing of a list of scrape configs (in the format expected in
// a Prometheus config map or additionalScrapeConfigs secret)
// GIVEN an updated scrape config job and its name
// WHEN the function is called
// THEN the scrape config job should be either added to the scrape configs list, or updated if a
//      job with that name already exists.
func TestEditScrapeJob(t *testing.T) {
	tests := []struct {
		name           string
		editJobName    string
		editConfigData string
		expectAdd      bool // true if new job should be added, false if it's an existing job
		expectRemove   bool // true if existing job should be removed
	}{
		{"add new job", NEW_JOB_NAME, NEW_SCRAPE_JOB, true, false},
		{"edit existing job", EXISTING_JOB_NAME, REPLACE_EXISTING_SCRAPE_JOB, false, false},
		{"remove job", EXISTING_JOB_NAME, "", false, true},
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
			origNumScrapeJobs := len(scrapejobs.Children())
			updatedScrapeJobs, err := EditScrapeJob(scrapejobs, tt.editJobName, newScrapeConfig)
			foundJobIndex := findScrapeJob(updatedScrapeJobs, tt.editJobName)
			if tt.expectAdd {
				// should have been added as a new job
				assert.Equal(t, origNumScrapeJobs+1, len(updatedScrapeJobs.Children()))
			} else if tt.expectRemove {
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

		})
	}
}
