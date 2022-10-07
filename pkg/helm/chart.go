// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

// ChartInfo contains Helm Chart.yaml data
type ChartInfo struct {
	APIVersion  string
	Description string
	Name        string
	Version     string
	AppVersion  string
}

// ChartInfoFnType - Package-level var and functions to allow overriding getReleaseState for unit test purposes
type ChartInfoFnType func(chartDir string) (ChartInfo, error)

var chartInfoFn ChartInfoFnType = getChartInfo

// SetChartInfoFunction Override the chart info function for unit testing
func SetChartInfoFunction(f ChartInfoFnType) {
	chartInfoFn = f
}

// SetDefaultChartInfoFunction Reset the chart info function
func SetDefaultChartInfoFunction() {
	chartInfoFn = getChartInfo
}

// GetChartInfo - public entry point
func GetChartInfo(chartDir string) (ChartInfo, error) {
	return chartInfoFn(chartDir)
}

// getChartInfo loads the Chart.yaml from the specified chart dir into a ChartInfo struct
func getChartInfo(chartDir string) (ChartInfo, error) {
	chartBytes, err := os.ReadFile(filepath.Join(chartDir + "/Chart.yaml"))
	if err != nil {
		return ChartInfo{}, err
	}
	chart := ChartInfo{}
	err = yaml.Unmarshal(chartBytes, &chart)
	if err != nil {
		return ChartInfo{}, err
	}
	return chart, nil
}
