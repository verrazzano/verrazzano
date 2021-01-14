// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"io/ioutil"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// ChartVersion is a Helm chart version struct
type ChartVersion struct {
	APIVersion  string
	Description string
	Name        string
	Version     string
	AppVersion  string
}

// For unit test purposes
var readFileFunction = ioutil.ReadFile

// GetChartVersion loads the Chart.yaml from the specified chart dir into a ChartVersion struct
func GetChartVersion(chartDir string) (ChartVersion, error) {
	chartBytes, err := readFileFunction(filepath.Join(chartDir + "/Chart.yaml"))
	if err != nil {
		return ChartVersion{}, err
	}
	chartVersion := ChartVersion{}
	err = yaml.Unmarshal(chartBytes, &chartVersion)
	if err != nil {
		return ChartVersion{}, err
	}
	return chartVersion, nil
}
