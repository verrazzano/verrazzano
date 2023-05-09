// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fs

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/helm"
	vcmtesthelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/tests/pkg/helpers"
)

const (
	testData      = "testdata"
	testChart     = "testChart"
	testVersion   = "x.y.z"
	testChartsDir = "testdata/charts"
)

// TestHelmChartFileSystem_RearrangeChartDirectory_CopyFailsThrowsError tests that function RearrangeChartDirectory fails when copying the chart fails
// GIVEN a call to RearrangeChartDirectory
//
//	WHEN copying the chart fails
//	THEN the rearrange operation returns error.
func TestHelmChartFileSystem_RearrangeChartDirectory_CopyFailsThrowsError(t *testing.T) {
	runner = vzos.GenericTestRunner{
		Err: fmt.Errorf(vcmtesthelpers.DummyError),
	}
	files := HelmChartFileSystem{}
	err := files.RearrangeChartDirectory(testChartsDir, testChart, testVersion)
	assert.ErrorContains(t, err, vcmtesthelpers.DummyError)
	runner = vzos.DefaultRunner{}
	os.RemoveAll(testData)
}

// TestHelmChartFileSystem_RearrangeChartDirectory_NoError tests that function RearrangeChartDirectory passes
// GIVEN a call to RearrangeChartDirectory
//
//	WHEN none of the operation fails
//	THEN the rearrange operation succeeds.
func TestHelmChartFileSystem_RearrangeChartDirectory_NoError(t *testing.T) {
	runner = vzos.GenericTestRunner{}
	files := HelmChartFileSystem{}
	err := files.RearrangeChartDirectory(testChartsDir, testChart, "a.b.c")
	assert.NoError(t, err)
	runner = vzos.DefaultRunner{}
	os.RemoveAll(testData)
}

// TestHelmChartFileSystem_SaveUpstreamChart_NoError tests that function SaveUpstreamChart passes
// GIVEN a call to SaveUpstreamChart
//
//	WHEN none of the operation fails
//	THEN the save upstream operation succeeds.
func TestHelmChartFileSystem_SaveUpstreamChart_NoError(t *testing.T) {
	runner = vzos.GenericTestRunner{}
	files := HelmChartFileSystem{}
	err := files.SaveUpstreamChart(testChartsDir, testChart, testVersion, testVersion)
	assert.NoError(t, err)
	runner = vzos.DefaultRunner{}
	os.RemoveAll(testData)
}

// TestHelmChartFileSystem_SaveUpstreamChart_NoError tests that function SaveUpstreamChart fails when copying fails
// GIVEN a call to SaveUpstreamChart
//
//	WHEN copying chart to upstream fails
//	THEN the save upstream operation fails.
func TestHelmChartFileSystem_SaveUpstreamChart_Error(t *testing.T) {
	runner = vzos.GenericTestRunner{
		Err: fmt.Errorf(vcmtesthelpers.DummyError),
	}
	files := HelmChartFileSystem{}
	err := files.SaveUpstreamChart(testChartsDir, testChart, testVersion, testVersion)
	assert.ErrorContains(t, err, vcmtesthelpers.DummyError)
	runner = vzos.DefaultRunner{}
	os.RemoveAll(testData)
}

// TestHelmChartFileSystem_SaveChartProvenance_NoError tests that function SaveChartProvenance succeeds for valid chart provenance data
// GIVEN a call to SaveChartProvenance
//
//	WHEN chart provenance is valid
//	THEN the save chart provenance passes.
func TestHelmChartFileSystem_SaveChartProvenance_NoError(t *testing.T) {
	files := HelmChartFileSystem{}
	err := files.SaveChartProvenance(testChartsDir, &helm.ChartProvenance{}, testChart, testVersion)
	assert.NoError(t, err)
	os.RemoveAll(testData)
}
