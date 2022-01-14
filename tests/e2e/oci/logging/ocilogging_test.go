// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/common/auth"
	"github.com/oracle/oci-go-sdk/v53/loggingsearch"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	compartmentIDEnvVar = "COMPARTMENT_ID"
	logGroupIDEnvVar    = "LOG_GROUP_ID"
	nsLogIDEnvVar       = "NS_LOG_ID"
	ociRegionEnvVar     = "OCI_CLI_REGION"

	waitTimeout     = 10 * time.Minute
	pollingInterval = 30 * time.Second
)

var compartmentID string
var logGroupID string
var nsLogID string
var region string
var logSearchClient loggingsearch.LogSearchClient

var failed = false

var t = framework.NewTestFramework("logging")

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {
	compartmentID = os.Getenv(compartmentIDEnvVar)
	logGroupID = os.Getenv(logGroupIDEnvVar)
	nsLogID = os.Getenv(nsLogIDEnvVar)
	region = os.Getenv(ociRegionEnvVar)
	Expect(compartmentID).ToNot(BeEmpty(), fmt.Sprintf("%s env var must be set", compartmentIDEnvVar))
	Expect(logGroupID).ToNot(BeEmpty(), fmt.Sprintf("%s env var must be set", logGroupIDEnvVar))
	Expect(nsLogID).ToNot(BeEmpty(), fmt.Sprintf("%s env var must be set", nsLogIDEnvVar))
	// region is optional so don't Expect

	var err error
	logSearchClient, err = getLogSearchClient(region)
	Expect(err).ShouldNot(HaveOccurred(), "Error configuring OCI SDK client")
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	pkg.Concurrently(
		func() {
			start := time.Now()
			pkg.UndeploySpringBootApplication()
			metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
		},
		func() {
			start := time.Now()
			pkg.UndeployHelloHelidonApplication()
			metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
		},
	)
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("OCI Logging", func() {
	var systemLogID, defaultAppLogID string

	t.BeforeEach(func() {
		var err error
		systemLogID, defaultAppLogID, err = getLogIdentifiersFromVZCustomResource()
		Expect(err).ShouldNot(HaveOccurred())
		Expect(systemLogID).ToNot(BeEmpty())
		Expect(defaultAppLogID).ToNot(BeEmpty())
	})

	t.Context("initially", func() {
		// GIVEN a Verrazzano installation
		// WHEN I search for log records in the kube-system namespace
		// THEN I expect to find at least one record
		t.It("the system log object has recent log records in the kube-system namespace", func() {
			Eventually(func() (int, error) {
				logs, err := getLogRecordsFromOCI(&logSearchClient, compartmentID, logGroupID, systemLogID, "kube-system")
				if err != nil {
					return 0, err
				}
				return *logs.Summary.ResultCount, nil
			}, waitTimeout, pollingInterval).Should(Not(BeZero()), "Expected to find kube-system logs but found none")
		})

		// GIVEN a Verrazzano installation
		// WHEN I search for log records in the verrazzano-system namespace
		// THEN I expect to find at least one record
		t.It("the system log object has recent log records in the verrazzano-system namespace", func() {
			Eventually(func() (int, error) {
				logs, err := getLogRecordsFromOCI(&logSearchClient, compartmentID, logGroupID, systemLogID, constants.VerrazzanoSystemNamespace)
				if err != nil {
					return 0, err
				}
				return *logs.Summary.ResultCount, nil
			}, waitTimeout, pollingInterval).Should(Not(BeZero()), "Expected to find verrazzano-system logs but found none")
		})

		// GIVEN a Verrazzano installation with no applications installed
		// WHEN I search for log records in the default app Log object
		// THEN I expect to find no records
		t.It("the default app log object has no log records", func() {
			logs, err := getLogRecordsFromOCI(&logSearchClient, compartmentID, logGroupID, defaultAppLogID, "")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*logs.Summary.ResultCount).To(BeZero(), "Expected no default app logs but found at least one")
		})

		// GIVEN a Verrazzano installation with no applications installed
		// WHEN I search for log records in the namespace-specific app Log object
		// THEN I expect to find no records
		t.It("the namespace-specific app log object has no log records", func() {
			logs, err := getLogRecordsFromOCI(&logSearchClient, compartmentID, logGroupID, nsLogID, "")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*logs.Summary.ResultCount).To(BeZero(), "Expected no namespace-specific app logs but found at least one")
		})
	})

	t.Context("after deploying an example application", func() {
		// GIVEN a Verrazzano installation with an application deployed
		// WHEN I search for log records in the default app Log object using the application namespace
		// THEN I expect to find at least one record
		t.It("the default app log object has recent log records", func() {

			start := time.Now()
			pkg.DeploySpringBootApplication()
			metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

			Eventually(func() (int, error) {
				logs, err := getLogRecordsFromOCI(&logSearchClient, compartmentID, logGroupID, defaultAppLogID, pkg.SpringbootNamespace)
				if err != nil {
					return 0, err
				}
				return *logs.Summary.ResultCount, nil
			}, waitTimeout, pollingInterval).Should(Not(BeZero()))
		})
	})

	t.Context("after deploying an example application where the namespace overrides the default app OCI Log id", func() {
		// GIVEN a Verrazzano installation
		// WHEN I deploy an example application and annotate the namespace with an OCI Log id
		// AND I search for log records in that log object
		// THEN I expect to find at least one record
		t.It("the namespace-specific app log object has recent log records", func() {

			start := time.Now()
			pkg.DeployHelloHelidonApplication(nsLogID)
			metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

			Eventually(func() (int, error) {
				logs, err := getLogRecordsFromOCI(&logSearchClient, compartmentID, logGroupID, nsLogID, pkg.HelloHelidonNamespace)
				if err != nil {
					return 0, err
				}
				return *logs.Summary.ResultCount, nil
			}, waitTimeout, pollingInterval).Should(Not(BeZero()))
		})
	})
})

// getLogIdentifiersFromVZCustomResource returns the system and default app OCI Log identifiers from the
// Verrazzano custom resource.
func getLogIdentifiersFromVZCustomResource() (string, string, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		return "", "", err
	}
	vz, err := pkg.GetVerrazzanoInstallResourceInCluster(kubeconfigPath)
	if err != nil {
		return "", "", err
	}
	if vz.Spec.Components.Fluentd.OCI == nil {
		return "", "", fmt.Errorf("expected to find Fluentd OCI logging settings but found nil")
	}

	return vz.Spec.Components.Fluentd.OCI.SystemLogID, vz.Spec.Components.Fluentd.OCI.DefaultAppLogID, nil
}

// getLogRecordsFromOCI searches an OCI Log object for log records in the last 15 minutes. If the optional
// namespace is specified, only log records in the namespace are matched, otherwise search for all log records
// in the Log object identified by the compartment id, log group id, and log id.
func getLogRecordsFromOCI(client *loggingsearch.LogSearchClient, compartmentID, logGroupID, logID, namespace string) (*loggingsearch.SearchLogsResponse, error) {
	pkg.Log(pkg.Info, "Checking for recent log records")

	var query string
	if namespace == "" {
		// no namespace specified, so fetch all records in the Log object
		query = `search "%s/%s/%s" | sort by datetime desc`
		query = fmt.Sprintf(query, compartmentID, logGroupID, logID)
	} else {
		query = `search "%s/%s/%s" | where "data"."kubernetes.namespace_name"='%s' | sort by datetime desc`
		query = fmt.Sprintf(query, compartmentID, logGroupID, logID, namespace)
	}

	now := time.Now()
	past := now.Add(-time.Minute * 15)
	search := loggingsearch.SearchLogsDetails{
		TimeStart:   &common.SDKTime{Time: past},
		TimeEnd:     &common.SDKTime{Time: now},
		SearchQuery: &query,
	}

	pkg.Log(pkg.Debug, fmt.Sprintf("Running log search query: %s", query))
	logs, err := client.SearchLogs(context.Background(), loggingsearch.SearchLogsRequest{SearchLogsDetails: search})
	if err != nil {
		return nil, err
	}

	pkg.Log(pkg.Debug, fmt.Sprintf("Found %d log records", *logs.Summary.ResultCount))
	if *logs.Summary.ResultCount > 0 {
		pkg.Log(pkg.Debug, fmt.Sprintf("Last record: %s", logs.Results[0].String()))
	}

	return &logs, nil
}

// getLogSearchClient returns an OCI SDK client for searching logs. If a region is specified then
// use an instance principal auth provider, otherwise use the default provider (auth config comes from
// an OCI config file or environment variables). Instance principals are used when running in the
// CI/CD pipelines while the default provider is suitable for running locally.
func getLogSearchClient(region string) (loggingsearch.LogSearchClient, error) {
	var provider common.ConfigurationProvider
	var err error

	if region != "" {
		pkg.Log(pkg.Info, fmt.Sprintf("Using OCI SDK instance principal provider with region: %s", region))
		provider, err = auth.InstancePrincipalConfigurationProviderForRegion(common.StringToRegion(region))
	} else {
		pkg.Log(pkg.Info, "Using OCI SDK default provider")
		provider = common.DefaultConfigProvider()
	}

	if err != nil {
		return loggingsearch.LogSearchClient{}, err
	}

	return loggingsearch.NewLogSearchClientWithConfigurationProvider(provider)
}
