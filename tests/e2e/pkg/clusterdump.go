// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test"
)

//ClusterDumpWrapper creates cluster snapshots if the test fails (spec or aftersuite)
// A maximum of two cluster snapshots will be generated:
// - a snapshot if any spec in the suite fails
// - a snapshot if the aftersuite fails
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
	return ginkgo.AfterSuite(func() {
		if c.failed || !c.beforeSuitePassed {
			ExecuteBugReportWithDirectorySuffix(fmt.Sprintf("fail-%d", time.Now().Unix()), c.namespaces...)
		}

		// ginkgo.Fail and gomega matchers panic if they fail. Recover is used to capture the panic and
		// generate the cluster snapshot
		defer func() {
			if r := recover(); r != nil {
				ExecuteBugReportWithDirectorySuffix(fmt.Sprintf("aftersuite-%d", time.Now().Unix()), c.namespaces...)
			}
		}()
		body()
	})
}

// ExecuteClusterDump executes the cluster dump tool.
// clusterDumpCommand - The fully qualified cluster dump executable.
// kubeconfig - The kube config file to use when executing the cluster dump tool.
// clusterDumpDirectory - The directory to store the cluster dump within.
func ExecuteClusterDump(clusterDumpCommand string, kubeconfig string, clusterDumpDirectory string) error {
	var cmd *exec.Cmd
	fmt.Printf("Execute cluster dump: KUBECONFIG=%s; %s -d %s\n", kubeconfig, clusterDumpCommand, clusterDumpDirectory)
	if clusterDumpCommand == "" {
		return nil
	}
	reportFile := fmt.Sprintf("%s/cluster-snapshot/analysis.report", clusterDumpDirectory)
	cmd = exec.Command(clusterDumpCommand, "-d", clusterDumpDirectory, "-r", reportFile)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
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
// kubeconfig - The kube config file to use when executing the bug-report CLI.
// bugReportDirectory - The directory to store the bug report within.
// ns - One or more additional namespaces, from where the resources need to be captured by the bug-report CLI
func executeBugReport(vzCommand string, kubeconfig string, bugReportDirectory string, ns ...string) error {
	fmt.Println("Bug report called with namespace(s) ", ns)
	var cmd *exec.Cmd
	if vzCommand == "" {
		return nil
	}

	filename := fmt.Sprintf("%s/%s", bugReportDirectory, "bug-report.tar.gz")
	os.MkdirAll(bugReportDirectory, 0755)

	if len(ns) > 0 {
		includeNS := strings.Join(ns[:], ",")
		cmd = exec.Command(vzCommand, "bug-report", "--report-file", filename, "--include-namespaces", includeNS)
	} else {
		cmd = exec.Command(vzCommand, "bug-report", "--report-file", filename)
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
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
	err := analyzeBugReport(vzCommand, bugReportDirectory, "bug-report.tar.gz")
	if err != nil {
		return err
	}
	return nil
}

// ExecuteBugReportWithDirectorySuffix executes the bug-report CLI.
// directorySuffix - The suffix for the directory where the bug-report CLI needs to create the report file.
// ns - One or more additional namespaces, from where the resources need to be captured by the bug-report CLI
func ExecuteBugReportWithDirectorySuffix(directorySuffix string, ns ...string) error {
	kubeconfig := os.Getenv("DUMP_KUBECONFIG")
	bugReportDirectory := filepath.Join(os.Getenv("DUMP_DIRECTORY")+"/bug-report", directorySuffix)
	vzCommand := os.Getenv("VZ_COMMAND")
	return executeBugReport(vzCommand, kubeconfig, bugReportDirectory, ns...)
}

// ExecuteBugReport executes the cluster bug-report CLI using config from environment variables.
// DUMP_KUBECONFIG - The kube config file to use when executing the bug-report CLI.
// DUMP_DIRECTORY - The directory to store the cluster snapshot within.
// DUMP_COMMAND - The fully qualified cluster snapshot executable.
// ns - One or more additional namespaces, from where the resources need to be captured by the bug-report CLI
func ExecuteBugReport(ns ...string) error {
	err := ExecuteBugReportWithDirectorySuffix("", ns...)
	if err != nil {
		return err
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
	kubeconfig := os.Getenv(test.DumpKubeconfigEnvVarName)

	containerPath := fmt.Sprintf("%s/%s:%s", namespace, podName, containerLogsDir)
	destDir := fmt.Sprintf("%s/%s/%s", directory, podName, containerName)

	cmd := exec.Command("kubectl", "cp", containerPath, "-c", containerName, destDir)
	Log(Info, fmt.Sprintf("kubectl command to capture %s logs: %s", podName, cmd.String()))
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		Log(Info, fmt.Sprintf("Error START kubectl %s end log copy, err: %s", podName, err))
	}
	if err := cmd.Wait(); err != nil {
		Log(Info, fmt.Sprintf("Error WAIT kubectl %s end log copy, err: %s", podName, err))
	}
}

// analyzeBugReport extracts the bug report and runs vz analyze by providing the extracted directory for flag --capture-dir
func analyzeBugReport(vzCommand, bugReportDirectory, bugReportFile string) error {
	cmd := exec.Command("tar", "-xf", bugReportFile)
	cmd.Dir = bugReportDirectory
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start the command to extract the bug report %v \n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed waiting for the command to extract the bug report %v \n", err)
		return err
	}

	// Safe to remove bugReportFile
	os.Remove(filepath.Join(bugReportDirectory, bugReportFile))
	reportFile := fmt.Sprintf("%s/cluster-snapshot/analysis.report", bugReportDirectory)
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
