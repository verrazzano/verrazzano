// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test"
)

const (
	AnalysisReport  = "analysis.report"
	BugReport       = "bug-report.tar.gz"
	FullCluster     = "full-cluster"
	ClusterSnapshot = "cluster-snapshot"
)

//ClusterDumpWrapper creates cluster snapshots if the test fails (spec or aftersuite)
// A maximum of two cluster snapshots will be generated:
// - snapshot if any spec in the suite fails
// - snapshot if the aftersuite fails
type ClusterDumpWrapper struct {
	failed            bool
	beforeSuitePassed bool
	namespaces        []string
}

func NewClusterDumpWrapper(ns ...string) *ClusterDumpWrapper {
	clusterDump := ClusterDumpWrapper{namespaces: ns}
	return &clusterDump
}

func (c *ClusterDumpWrapper) BeforeSuite(body func()) bool {
	return ginkgo.BeforeSuite(func() {
		body()
		c.beforeSuitePassed = true
	})
}

//AfterEach wraps ginkgo.AfterEach
// usage: var _ = c.AfterEach(func() { ...after each logic... })
func (c *ClusterDumpWrapper) AfterEach(body func()) bool {
	return ginkgo.AfterEach(func() {
		c.failed = c.failed || ginkgo.CurrentSpecReport().Failed()
		body()
	})
}

//AfterSuite wraps ginkgo.AfterSuite
// usage: var _ = c.AfterSuite(func() { ...after suite logic... })
func (c *ClusterDumpWrapper) AfterSuite(body func()) bool {
	// Capture full cluster snapshot when environment variable CAPTURE_FULL_CLUSTER is set
	isFullCapture := os.Getenv("CAPTURE_FULL_CLUSTER")
	return ginkgo.AfterSuite(func() {
		if c.failed || !c.beforeSuitePassed {
			dirSuffix := fmt.Sprintf("fail-%d", time.Now().Unix())
			if strings.EqualFold(isFullCapture, "true") {
				executeClusterDumpWithEnvVarSuffix(dirSuffix)
			}
			executeBugReportWithDirectorySuffix(dirSuffix, c.namespaces...)
		}

		// ginkgo.Fail and gomega matchers panic if they fail. Recover is used to capture the panic and
		// generate the cluster snapshot
		defer func() {
			if r := recover(); r != nil {
				dirSuffix := fmt.Sprintf("aftersuite-%d", time.Now().Unix())
				if strings.EqualFold(isFullCapture, "true") {
					executeClusterDumpWithEnvVarSuffix(dirSuffix)
				}
				executeBugReportWithDirectorySuffix(dirSuffix, c.namespaces...)
			}
		}()
		body()
	})
}

// executeClusterDump executes the cluster dump tool.
// clusterDumpCommand - The fully qualified cluster dump executable.
// kubeConfig - The kube config file to use when executing the cluster dump tool.
// clusterDumpDirectory - The directory to store the cluster dump within.
func executeClusterDump(clusterDumpCommand string, kubeConfig string, clusterDumpDirectory string) error {
	var cmd *exec.Cmd
	fmt.Printf("Execute cluster dump: KUBECONFIG=%s; %s -d %s\n", kubeConfig, clusterDumpCommand, clusterDumpDirectory)
	if clusterDumpCommand == "" {
		return nil
	}
	reportFile := filepath.Join(clusterDumpDirectory, AnalysisReport)
	if err := os.MkdirAll(clusterDumpDirectory, 0755); err != nil {
		return err
	}

	cmd = exec.Command(clusterDumpCommand, "-d", clusterDumpDirectory, "-r", reportFile)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

// executeBugReport executes the bug-report CLI to capture the cluster resources and analyze on the bug report
// vzCommand - The fully qualified bug report executable.
// kubeConfig - The kube config file to use when executing the bug-report CLI.
// bugReportDirectory - The directory to store the bug report within.
// ns - One or more additional namespaces, from where the resources need to be captured by the bug-report CLI
func executeBugReport(vzCommand string, kubeConfig string, bugReportDirectory string, ns ...string) error {
	var cmd *exec.Cmd
	if vzCommand == "" {
		return nil
	}

	filename := filepath.Join(bugReportDirectory, BugReport)
	if err := os.MkdirAll(bugReportDirectory, 0755); err != nil {
		return err
	}

	if len(ns) > 0 {
		includeNS := strings.Join(ns[:], ",")
		cmd = exec.Command(vzCommand, "bug-report", "--report-file", filename, "--include-namespaces", includeNS)
	} else {
		cmd = exec.Command(vzCommand, "bug-report", "--report-file", filename)
	}
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start the command bug-report: %v \n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed waiting for the command bug-report: %v \n", err)
		return err
	}
	// Extract the bug-report and run vz-analyze
	err := analyzeBugReport(kubeConfig, vzCommand, bugReportDirectory)
	if err != nil {
		return err
	}
	return nil
}

// analyzeBugReport extracts the bug report and runs vz analyze by providing the extracted directory for flag --capture-dir
func analyzeBugReport(kubeConfig, vzCommand, bugReportDirectory string) error {
	bugReportFile := filepath.Join(bugReportDirectory, BugReport)

	cmd := exec.Command("tar", "-xf", bugReportFile, "-C", bugReportDirectory)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfig))

	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start the command to extract the bug report %v \n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed waiting for the command to extract the bug report %v \n", err)
		return err
	}

	// Safe to remove bugReportFile
	os.Remove(filepath.Join(bugReportFile))
	reportFile := filepath.Join(bugReportDirectory, ClusterSnapshot, AnalysisReport)
	cmd = exec.Command(vzCommand, "analyze", "--capture-dir", bugReportDirectory, "--report-format", "detailed", "--report-file", reportFile)
	fmt.Println(cmd.String())
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start the command analyze %v \n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed waiting for the command analyze %v \n", err)
		return err
	}
	return nil
}

func executeClusterDumpWithEnvVarSuffix(directorySuffix string) error {
	kubeConfig := os.Getenv("DUMP_KUBECONFIG")
	clusterDumpDirectory := filepath.Join(os.Getenv("DUMP_DIRECTORY"), directorySuffix, FullCluster)
	clusterDumpCommand := os.Getenv("DUMP_COMMAND")
	return executeClusterDump(clusterDumpCommand, kubeConfig, clusterDumpDirectory)
}

// executeBugReportWithDirectorySuffix executes the bug-report CLI.
// directorySuffix - The suffix for the directory where the bug-report CLI needs to create the report file.
// ns - One or more additional namespaces, from where the resources need to be captured by the bug-report CLI
func executeBugReportWithDirectorySuffix(directorySuffix string, ns ...string) error {
	kubeConfig := os.Getenv("DUMP_KUBECONFIG")
	bugReportDirectory := filepath.Join(os.Getenv("DUMP_DIRECTORY"), directorySuffix)
	vzCommand := os.Getenv("VZ_COMMAND")
	return executeBugReport(vzCommand, kubeConfig, bugReportDirectory, ns...)
}

// ExecuteBugReport executes the cluster bug-report CLI using config from environment variables.
// DUMP_KUBECONFIG - The kube config file to use when executing the bug-report CLI.
// DUMP_DIRECTORY - The directory to store the cluster snapshot within.
// DUMP_COMMAND - The fully qualified cluster snapshot executable.
// One or more additional namespaces specified using ns are set for the flag --include-namespaces
func ExecuteBugReport(ns ...string) error {
	var err1, err2 error
	// Capture full cluster snapshot when environment variable CAPTURE_FULL_CLUSTER is set
	isFullCapture := os.Getenv("CAPTURE_FULL_CLUSTER")
	if strings.EqualFold(isFullCapture, "true") {
		err1 = executeClusterDumpWithEnvVarSuffix("")
	}
	err2 = executeBugReportWithDirectorySuffix("", ns...)
	cumulativeError := ""
	if err1 != nil || err2 != nil {
		if err1 != nil {
			cumulativeError += err1.Error() + ";"
		}
		if err2 != nil {
			cumulativeError += err2.Error() + ";"
		}
		return errors.New(cumulativeError)
	}
	return nil
}

// CaptureContainerLogs executes a "kubectl cp" command to copy a container's log directories to a local path on disk for examination.
// This utilizes the cluster snapshot directory and KUBECONFIG settings do capture the logs to the same path as the cluster snapshot location;
// the container log directory is copied to DUMP_DIRECTORY/podName.
//
// namespace - The namespace of the target pod
// podName - The name of the pod
// containerName - The target container name within the pod
// containerLogsDir - The logs directory location within the container
//
// DUMP_KUBECONFIG - The kube config file to use when executing the bug-report CLI.
// DUMP_DIRECTORY - The directory to store the cluster snapshot within.
func CaptureContainerLogs(namespace string, podName string, containerName string, containerLogsDir string) {
	directory := os.Getenv(test.DumpDirectoryEnvVarName)
	kubeConfig := os.Getenv(test.DumpKubeconfigEnvVarName)

	containerPath := fmt.Sprintf("%s/%s:%s", namespace, podName, containerLogsDir)
	destDir := fmt.Sprintf("%s/%s/%s", directory, podName, containerName)

	cmd := exec.Command("kubectl", "cp", containerPath, "-c", containerName, destDir)
	Log(Info, fmt.Sprintf("kubectl command to capture %s logs: %s", podName, cmd.String()))
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeConfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		Log(Info, fmt.Sprintf("Error START kubectl %s end log copy, err: %s", podName, err))
	}
	if err := cmd.Wait(); err != nil {
		Log(Info, fmt.Sprintf("Error WAIT kubectl %s end log copy, err: %s", podName, err))
	}
}
