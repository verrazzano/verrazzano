// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"flag"
	"fmt"
	"github.com/joshdk/go-junit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"log"
	"regexp"
	"strings"
)

// Flags
var promCredentials string
var prometheusURL string
var testReportDir string
var gitCommit string
var testEnvironment string
var gitBranch string
var buildNumber string
var buildJobName string

// Credentials to push the metrics
var user = ""
var pwd = ""

// instance
var inst = ""

// constants
const commitSha = "commit_sha"
const branch = "branch"
const jobNumber = "job_number"
const testEnv = "test_env"
const instance = "instance"
const testSuite = "test_suite"
const metricPrefix = "_verrazzano_"
const statusSuffix = "_status"
const timeSuffix = "_time"

func main() {
	processInput()
	processJunitReports()
}

// Process the input arguments and validate
func processInput() (exitCode int) {
	flag.StringVar(&testReportDir, "report-dir", "", "The directory containing the JUnit reports")
	flag.StringVar(&promCredentials, "prometheus-credential", "", "Prometheus credentials")
	flag.StringVar(&prometheusURL, "prometheus-url", "", "Prometheus Push Gateway URL")
	flag.StringVar(&gitCommit, "commit-sha", "", "Git commit sha")
	flag.StringVar(&testEnvironment, "test-env", "", "Test environment")
	flag.StringVar(&gitBranch, "branch-name", "", "Branch Name")
	flag.StringVar(&buildNumber, "build-number", "", "Build Number")
	flag.StringVar(&buildJobName, "job-name", "", "Job Name")

	flag.Parse()

	if testReportDir == "" {
		fmt.Printf("\nRequired flag report-dir is not specified, exiting.\n")
		printUsage()
		return 1
	}

	if promCredentials == "" {
		fmt.Printf("\nRequired flag prometheus-credential is not specified, exiting.\n")
		printUsage()
		return 1
	}

	if prometheusURL == "" {
		fmt.Printf("\nRequired flag prometheus-url is not specified, exiting.\n")
		printUsage()
		return 1
	}
	prometheusURL = strings.TrimSuffix(prometheusURL, "/")

	if gitCommit == "" {
		fmt.Printf("\nRequired flag commit-sha hash not specified, exiting.\n")
		printUsage()
		return 1
	}

	if testEnvironment == "" {
		fmt.Printf("\nRequired flag test-env is not specified, exiting.\n")
		printUsage()
		return 1
	}

	if gitBranch == "" {
		fmt.Printf("\nRequired flag branch-name is not specified, exiting.\n")
		printUsage()
		return 1
	}

	if buildNumber == "" {
		fmt.Printf("\nRequired flag build-number is not specified, exiting.\n")
		printUsage()
		return 1
	}

	if buildJobName == "" {
		fmt.Printf("\nRequired flag job-name is not specified, exiting.\n")
		printUsage()
		return 1
	}
	// Extract only the first part, and remove the feature branch
	jobParts := strings.Split(buildJobName, "/")
	buildJobName = jobParts[0]

	// extract user and password from the promCredentials
	cred := strings.Split(promCredentials, ":")
	user = cred[0]
	pwd = cred[1]

	inst = removeSpecialChars(gitBranch)
	return 0
}

// Process the Junit reports created by the tests, recursively under the directory testReportDir
func processJunitReports() {
	suites, err := junit.IngestDir(testReportDir)
	if err != nil {
		log.Fatalf("failed to ingest JUnit xml %v", err)
	}
	var testStatus float64
	for _, suite := range suites {
		if suite.Totals.Tests == suite.Totals.Passed {
			testStatus = 1.0
		} else {
			testStatus = -1.0
		}
		metricName := removeSpecialChars(suite.Name)
		emitTestMetrics(metricName, statusSuffix, testStatus)
		emitTestMetrics(metricName, timeSuffix, float64(suite.Totals.Duration.Milliseconds()))
	}
}

// Emit metrics for the test status and execution time
func emitTestMetrics(metricName string, metricSuffix string, metricValue float64) {
	metricToEmit := metricPrefix+ metricName + metricSuffix
	testMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: metricToEmit,
		ConstLabels: prometheus.Labels{
			commitSha: gitCommit,
			branch:    gitBranch,
			jobNumber: buildNumber,
			testEnv:   testEnvironment,
			instance:  inst,
			testSuite: metricName,
		},
	})
	testMetric.SetToCurrentTime()
	testMetric.Set(metricValue)
	if err := push.New(prometheusURL, buildJobName).
		Collector(testMetric).
		BasicAuth(user, pwd).
		Add(); err != nil {
		fmt.Println("Could not push completion time to push gateway, ", err)
		log.Fatal(err)
	}
	fmt.Printf("Successfully pushed metric %v\n", metricToEmit)
}

// The label instance and the value for metric doesn't allow special characters.
// Replace all the non-alphanumeric characters with an underscore
func removeSpecialChars(inputParam string) string {
	returnVal := strings.ReplaceAll(strings.ToLower(inputParam), " ", "_")
	reg, err := regexp.Compile("[^a-zA-Z0-9/_]")
	if err != nil {
		log.Fatal(err)
	}
	returnVal = reg.ReplaceAllString(returnVal, "")
	returnVal = strings.ReplaceAll(strings.ToLower(returnVal), "/", "_")
	return returnVal
}

func printUsage() {
	usageString := "Usage: go run main.go --report-dir=<directory containing JUnit reports> " +
		"--prometheus-credential=<credentials to push the metrics in user:password format> " +
		"--prometheus-url=<Prometheus Push Gateway URL> --commit-sha=<Git commit hash> --test-env=<test environment> " +
		"--branch-name=<name of the Git branch> --build-number=<Build Number>" +
		"--job-name=<CI job name>"
	fmt.Println(usageString)
}
