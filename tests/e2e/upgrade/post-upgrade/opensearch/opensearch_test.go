// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"encoding/json"
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
	documentFile    = "testdata/upgrade/opensearch/document1.json"
)

var t = framework.NewTestFramework("opensearch")

var _ = t.BeforeSuite(func() {})
var _ = t.AfterSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Post upgrade OpenSearch", Label("f:observability.logging.es"), func() {
	// It Wrapper to only run spec if component is supported on the current Verrazzano installation
	MinimumVerrazzanoIt := func(description string, f interface{}) {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
			Fail(err.Error())
		}
		supported, err := pkg.IsVerrazzanoMinVersion("1.3.0", kubeconfigPath)
		if err != nil {
			Fail(err.Error())
		}
		// Only run tests if Verrazzano is at least version 1.3.0
		if supported {
			t.It(description, f)
		} else {
			pkg.Log(pkg.Info, fmt.Sprintf("Skipping check '%v', Verrazzano is not at version 1.3.0", description))
		}
	}

	// GIVEN the OpenSearch pod
	// WHEN the indices are retrieved
	// THEN verify that they do not have the old indices
	MinimumVerrazzanoIt("Old indices are deleted", func() {
		Eventually(func() bool {
			kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
			if pkg.IsOpenSearchEnabled(kubeconfigPath) {
				oldIndicesPatterns := []string{"^verrazzano-namespace-.*$", "^verrazzano-systemd-journal$",
					"^verrazzano-logstash-.*$"}
				return pkg.IndicesNotExists(oldIndicesPatterns)
			}
			return true
		}, threeMinutes, pollingInterval).Should(BeTrue(), "Expected not to find any old indices")
	})

	// GIVEN the OpenSearch pod
	// WHEN the data streams are retrieved
	// THEN verify that they have data streams
	MinimumVerrazzanoIt("Data streams are created", func() {
		Eventually(func() bool {
			kubeconfigPath, _ := k8sutil.GetKubeConfigLocation()
			if pkg.IsOpenSearchEnabled(kubeconfigPath) {
				return pkg.CheckForDataStream("verrazzano-system")
			}
			return true
		}, threeMinutes, pollingInterval).Should(BeTrue(), "Expected not to find any old indices")
	})

	// GIVEN the OpenSearch pod
	// THEN verify that the data can be retrieved successfully
	t.It("OpenSearch get old data", func() {
		Eventually(func() bool {
			kubeConfigPath, _ := k8sutil.GetKubeConfigLocation()
			if pkg.IsOpenSearchEnabled(kubeConfigPath) {
				indexName := pkg.GetOpenSearchSystemIndex("verrazzano-system")
				file, err := pkg.FindTestDataFile(documentFile)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to find test data file: %v", err))
					return false
				}
				data, err := ioutil.ReadFile(file)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("failed to read test data file: %v", err))
					return false
				}
				var dataMap map[string]interface{}
				if err := json.Unmarshal(data, &dataMap); err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("OpenSearch: Error unmarshalling test document: %v", err))
				}
				query := pkg.ElasticQuery{
					Filters: []pkg.Match{
						{Key: "type", Value: dataMap["type"].(string)}},
					MustNot: []pkg.Match{},
				}
				result := pkg.SearchLog(fmt.Sprintf("%s/_doc", indexName), query)
				for k, v := range dataMap {
					if result[k] != v {
						pkg.Log(pkg.Error, fmt.Sprintf("Expected to have a document with field name %s and value %s", k, v))
						return false
					}
				}
			}
			return true
		}, threeMinutes, pollingInterval).Should(BeTrue(), "Expected to find the old data")
	})

	// GIVEN a VZ environment with
	// WHEN VZ custom resource is upgraded
	// THEN only the system logs that are as old as the retention period
	//      is migrated and older logs are purged
	t.It("OpenSearch system logs older than retention period is not available post upgrade", func() {
		systemRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy("system")
		if err != nil {
			Fail("Error getting retention period for system logs from VZ CR - " + err.Error())
		}
		oldLogsFound, err := pkg.ContainsDocsOlderThanRetentionPeriod("verrazzano-system", *systemRetentionPolicy.MinIndexAge)
		if err != nil {
			Fail("Error checking if docs older than retention period for system logs are present - " + err.Error())
		}
		Expect(oldLogsFound).To(Equal(false))
	})

	// GIVEN a VZ environment with
	// WHEN VZ custom resource is upgraded
	// THEN only the application logs that are as old as the retention period
	//      is migrated and older logs are purged
	t.It("OpenSearch application logs older than retention period is not available post upgrade", func() {
		applicationRetentionPolicy, err := pkg.GetVerrazzanoRetentionPolicy("application")
		if err != nil {
			Fail("Error getting retention period for system logs from VZ CR - " + err.Error())
		}
		applicationDataStreams, err := pkg.GetApplicationDataStreamNames()
		if err != nil {
			Fail("Error getting all application data stream names - " + err.Error())
		}
		for _, applicationDataStream := range applicationDataStreams {
			oldLogsFound, err := pkg.ContainsDocsOlderThanRetentionPeriod(applicationDataStream, *applicationRetentionPolicy.MinIndexAge)
			if err != nil {
				Fail("Error checking if older indices for application logs are present - " + err.Error())
			}
			Expect(oldLogsFound).To(Equal(false))
		}
	})

})
