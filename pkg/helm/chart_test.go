// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetChart tests getting the contents of Chart.yaml
// GIVEN a chart directory
//
//	WHEN I call GetChart
//	THEN the correct chart data is returned
func TestGetChart(t *testing.T) {
	assert := assert.New(t)
	chart, err := GetChartInfo("./testdata")
	assert.NoError(err, "GetChartInfo returned an error")
	assert.Equal(chart.APIVersion, "v1", "incorrect API version")
	assert.Equal(chart.Version, "0.8.0", "incorrect chart version")
	assert.Equal(chart.AppVersion, "0.8.0-app", "incorrect chart app version")
	assert.Equal(chart.Description, "Test Helm Chart", "incorrect chart description")
	assert.Equal(chart.Name, "testChart", "incorrect chart name")
}

// TestGetChartNoFile tests getting the error handling of getting a chart when the file doesn't exist
// GIVEN a chart directory with no chart file
//
//	WHEN I call GetChart
//	THEN an error is returned
func TestGetChartNoFile(t *testing.T) {
	assert := assert.New(t)
	_, err := GetChartInfo("./")
	assert.Error(err, "GetChartInfo should have returned an error")
}
