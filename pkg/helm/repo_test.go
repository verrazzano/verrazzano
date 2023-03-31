// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package helm

//const pocURL = "http://localhost:9080/vz/stable"
//const pocRepoName = "vz-stable"
//
//func TestListCharts(t *testing.T) {
//	//err := ListChartsInRepo(vzlog.DefaultLogger(), "vz-stable-poc", "http://localhost:9080/vz/stable")
//	//assert.NoError(t, err)
//	t.Log("TBD")
//}

//func TestChartTypeNotFound(t *testing.T) {
//	a := assert.New(t)
//	chartType, err := LookupChartType(vzlog.DefaultLogger(), pocRepoName, pocURL, "mysql-operator", "2.0.8")
//	a.Error(err)
//	a.Equal(platformapi.UnclassifiedChartType, chartType)
//}
//
//func TestChartTypeModuleFound(t *testing.T) {
//	a := assert.New(t)
//	chartType, err := LookupChartType(vzlog.DefaultLogger(), pocRepoName, pocURL, "mysql-dummy", "0.1.3")
//	if err != nil {
//		return
//	}
//	a.NoError(err)
//	a.Equal(platformapi.ModuleChartType, chartType)
//}
//
//func TestChartTypeOperatorFound(t *testing.T) {
//	a := assert.New(t)
//	chartType, err := LookupChartType(vzlog.DefaultLogger(), pocRepoName, pocURL, "mysql-operator", "2.0.12")
//	if err != nil {
//		return
//	}
//	a.NoError(err)
//	a.Equal(platformapi.OperatorChartType, chartType)
//}
//
//func TestFindNearestSupportingChartVersion(t *testing.T) {
//	a := assert.New(t)
//	chartVersion, err := FindNearestSupportingChartVersion(vzlog.DefaultLogger(), "mysql-operator", pocRepoName, pocURL, "2.0.0")
//	if err != nil {
//		return
//	}
//	a.NoError(err)
//	a.Equal("2.0.12", chartVersion)
//}
//
//func Test_findChartEntry(t *testing.T) {
//	indexFile, err := loadAndSortRepoIndexFile(pocRepoName, pocURL)
//	if err != nil {
//		t.Logf("Repo could not be loaded for %s at %s", pocRepoName, pocURL)
//		return
//	}
//	chartVersions := findChartEntry(indexFile, "mysql-operator")
//	for _, version := range chartVersions {
//		t.Logf("Chart name: %s, chart version: %s, annotations: %v", version.Name, version.Version, version.Annotations)
//	}
//}
//
//func Test_findSupportingChartVersion(t *testing.T) {
//	a := assert.New(t)
//	indexFile, err := loadAndSortRepoIndexFile(pocRepoName, pocURL)
//	//a.NoError(err)
//	if err != nil {
//		t.Logf("Repo could not be loaded for %s at %s", pocRepoName, pocURL)
//		return
//	}
//
//	chartName := "mysql-dummy"
//	platformVersion := "2.0.0"
//	chartVersion, err := findSupportingChartVersion(vzlog.DefaultLogger(), indexFile, chartName, platformVersion)
//	a.NoError(err)
//	a.NotNilf(chartVersion, "Supporting version for chart %s for platform version %s", chartName, platformVersion)
//}
//
//func Test_ApplyModuleDefinitions(t *testing.T) {
//	a := assert.New(t)
//	_, err := loadAndSortRepoIndexFile(pocRepoName, pocURL)
//	//a.NoError(err)
//	if err != nil {
//		t.Logf("Repo could not be loaded for %s at %s", pocRepoName, pocURL)
//		return
//	}
//
//	testClient := fake.NewClientBuilder().WithScheme(newScheme()).Build()
//	chartName := "mysql-dummy"
//	ApplyModuleDefinitions(vzlog.DefaultLogger(), testClient, chartName, "0.1.2", pocURL)
//	a.NoError(err)
//}
//
//func newScheme() *runtime.Scheme {
//	scheme := runtime.NewScheme()
//	_ = platformapi.AddToScheme(scheme)
//	_ = v1beta1.AddToScheme(scheme)
//	_ = v1alpha1.AddToScheme(scheme)
//	return scheme
//}
