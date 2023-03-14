// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package helm

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	platformapi "github.com/verrazzano/verrazzano/platform-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const pocURL = "http://localhost:9080/vz/stable"
const pocRepoName = "vz-stable"

func TestListCharts(t *testing.T) {
	//err := ListChartsInRepo(vzlog.DefaultLogger(), "vz-stable-poc", "http://localhost:9080/vz/stable")
	//assert.NoError(t, err)
	t.Log("TBD")
}

func Test_findChartEntry(t *testing.T) {
	indexFile, err := loadAndSortRepoIndexFile(pocRepoName, pocURL)
	if err != nil {
		t.Logf("Repo could not be loaded for %s at %s", pocRepoName, pocURL)
		return
	}
	chartVersions := findChartEntry(indexFile, "mysql-operator")
	for _, version := range chartVersions {
		t.Logf("Chart name: %s, chart version: %s, annotations: %v", version.Name, version.Version, version.Annotations)
	}
}

func Test_findSupportingChartVersion(t *testing.T) {
	a := assert.New(t)
	indexFile, err := loadAndSortRepoIndexFile(pocRepoName, pocURL)
	//a.NoError(err)
	if err != nil {
		t.Logf("Repo could not be loaded for %s at %s", pocRepoName, pocURL)
		return
	}

	chartName := "mysql-dummy"
	platformVersion := "2.0.0"
	chartVersion, err := findSupportingChartVersion(vzlog.DefaultLogger(), indexFile, chartName, platformVersion)
	a.NoError(err)
	a.NotNilf(chartVersion, "Supporting version for chart %s for platform version %s", chartName, platformVersion)
}

func Test_ApplyModuleDefinitions(t *testing.T) {
	a := assert.New(t)
	indexFile, err := loadAndSortRepoIndexFile(pocRepoName, pocURL)
	//a.NoError(err)
	if err != nil {
		t.Logf("Repo could not be loaded for %s at %s", pocRepoName, pocURL)
		return
	}

	testClient := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	platformVersion := "2.0.0"
	chartName := "mysql-dummy"
	ApplyModuleDefinitions(vzlog.DefaultLogger(), testClient, chartName, pocRepoName, pocURL, platformVersion)
	chartVersion, err := findSupportingChartVersion(vzlog.DefaultLogger(), indexFile, chartName, platformVersion)
	a.NoError(err)
	a.NotNilf(chartVersion, "Supporting version for chart %s for platform version %s", chartName, platformVersion)
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = platformapi.AddToScheme(scheme)
	_ = v1beta1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	return scheme
}
