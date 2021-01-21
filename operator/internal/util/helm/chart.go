// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"io/ioutil"
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

// GetChartInfo loads the Chart.yaml from the specified chart dir into a ChartInfo struct
func GetChartInfo(chartDir string) (ChartInfo, error) {
	chartBytes, err := ioutil.ReadFile(filepath.Join(chartDir + "/Chart.yaml"))
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
