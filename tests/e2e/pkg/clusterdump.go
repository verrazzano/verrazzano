// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test"
)

//ClusterDumpWrapper creates cluster dumps if the test fails (spec or aftersuite)
// A maximum of two cluster dumps will be generated:
// - dump if any spec in the suite fails
// - dump if the aftersuite fails
type ClusterDumpWrapper struct {
	failed            bool
	beforeSuitePassed bool
}

func NewClusterDumpWrapper() *ClusterDumpWrapper {
	return &ClusterDumpWrapper{}
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
			ExecuteClusterDumpWithEnvVarSuffix(fmt.Sprintf("fail-%d", time.Now().Unix()))
			ExecuteBugReportWithEnvVarSuffix(fmt.Sprintf("fail-%d", time.Now().Unix()))
			ExecuteExtractBugReportWithEnvVarSuffix(fmt.Sprintf("fail-%d", time.Now().Unix()))
		}

		// ginkgo.Fail and gomega matchers panic if they fail. Recover is used to capture the panic and
		// generate the cluster dump
		defer func() {
			if r := recover(); r != nil {
				ExecuteClusterDumpWithEnvVarSuffix(fmt.Sprintf("aftersuite-%d", time.Now().Unix()))
				ExecuteBugReportWithEnvVarSuffix(fmt.Sprintf("aftersuite-%d", time.Now().Unix()))
				ExecuteExtractBugReportWithEnvVarSuffix(fmt.Sprintf("aftersuite-%d", time.Now().Unix()))
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

// ExecuteBugReport executes the cluster dump tool.
// vzCommand - The fully qualified bug report executable.
// kubeconfig - The kube config file to use when executing the cluster dump tool.
// bugReportDirectory - The directory to store the bug report within.
func ExecuteBugReport(vzCommand string, kubeconfig string, bugReportDirectory string) error {
	var cmd *exec.Cmd
	if vzCommand == "" {
		return nil
	}

	filename := fmt.Sprintf("%s/%s", bugReportDirectory, "bug-report.tar.gz")
	fmt.Printf("Starting bug report command: KUBECONFIG=%s; %s bug-report --report-file %s\n", kubeconfig, vzCommand, filename)
	os.MkdirAll(bugReportDirectory, 0755)
	cmd = exec.Command(vzCommand, "bug-report", "--report-file", filename)
	fmt.Printf("past the exec.Command \n")

	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("About to call start \n")
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start command %v \n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed waiting for command %v \n", err)
		return err
	}
	fmt.Printf("command succeeded without error \n")

	return nil
}

//ExecuteExtractBugReport extracts bug-report tar file to directory
// bugReportDirectory - The directory to store the bug report within.
// kubeconfig - The kube config file to use when executing the cluster dump tool.
func ExecuteExtractBugReport(kubeconfig string, bugReportDirectory string) error {
	var cmd *exec.Cmd
	filename := fmt.Sprintf("%s/%s", bugReportDirectory, "bug-report.tar.gz")
	extractedBugReportDir := bugReportDirectory + "/bug-report"
	fmt.Printf("Starting bug report extract: KUBECONFIG=%s; tar -xvf %s -C %s", kubeconfig, filename, extractedBugReportDir)
	errExtractedBugReportDir := os.MkdirAll(extractedBugReportDir, 0755)
	if errExtractedBugReportDir != nil {
		return errExtractedBugReportDir
	}
	cmd = exec.Command("tar", "-xvf", filename, "-C", extractedBugReportDir)
	fmt.Printf("past the exec.Command \n")

	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("About to call start \n")
	if err := cmd.Start(); err != nil {
		fmt.Printf("Failed to start command %v \n", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Failed waiting for command %v \n", err)
		return err
	}
	fmt.Printf("command succeeded without error \n")
	return nil
}

func getKubeConfig() (string, error) {
	var kubeconfig string
	dumpKubeconfig := os.Getenv("DUMP_KUBECONFIG")
	vzKubeconfig := os.Getenv("VERRAZZANO_KUBECONFIG")
	if dumpKubeconfig == "" && vzKubeconfig == "" {
		return "", fmt.Errorf("error getting kubeconfig")
	} else if dumpKubeconfig != "" {
		kubeconfig = dumpKubeconfig
	} else {
		kubeconfig = vzKubeconfig
	}
	return kubeconfig, nil
}

func ExecuteClusterDumpWithEnvVarSuffix(directorySuffix string) error {
	kubeconfig, err := getKubeConfig()
	if err != nil {
		return err
	}
	clusterDumpDirectory := filepath.Join(os.Getenv("DUMP_DIRECTORY"), directorySuffix)
	clusterDumpCommand := os.Getenv("DUMP_COMMAND")
	return ExecuteClusterDump(clusterDumpCommand, kubeconfig, clusterDumpDirectory)
}

func ExecuteBugReportWithEnvVarSuffix(directorySuffix string) error {
	kubeconfig, err := getKubeConfig()
	if err != nil {
		return err
	}
	bugReportDirectory := filepath.Join(os.Getenv("DUMP_DIRECTORY")+"/bug-report", directorySuffix)
	vzCommand := os.Getenv("VZ_COMMAND")
	return ExecuteBugReport(vzCommand, kubeconfig, bugReportDirectory)
}

func ExecuteExtractBugReportWithEnvVarSuffix(directorySuffix string) error {
	kubeconfig, err := getKubeConfig()
	if err != nil {
		return err
	}
	bugReportDirectory := filepath.Join(os.Getenv("DUMP_DIRECTORY")+"/bug-report", directorySuffix)
	return ExecuteExtractBugReport(kubeconfig, bugReportDirectory)
}

// ExecuteClusterDumpWithEnvVarConfig executes the cluster dump tool using config from environment variables.
// DUMP_KUBECONFIG - The kube config file to use when executing the cluster dump tool.
// DUMP_DIRECTORY - The directory to store the cluster dump within.
// DUMP_COMMAND - The fully quallified cluster dump executable.
func ExecuteClusterDumpWithEnvVarConfig() error {
	err1 := ExecuteClusterDumpWithEnvVarSuffix("")
	err2 := ExecuteBugReportWithEnvVarSuffix("")
	err3 := ExecuteExtractBugReportWithEnvVarSuffix("")
	cumulativeError := ""
	if err1 != nil || err2 != nil || err3 != nil {
		if err1 != nil {
			cumulativeError += err1.Error() + ";"
		}
		if err2 != nil {
			cumulativeError += err2.Error() + ";"
		}
		if err3 != nil {
			cumulativeError += err3.Error() + ";"
		}
		return errors.New(cumulativeError)
	}
	return nil
}

// DumpContainerLogs executes a "kubectl cp" command to copy a container's log directories to a local path on disk for examination.
// This utilizes the cluster dump directory and KUBECONFIG settings do dump the logs to the same path as the cluster dump location;
// the container log directory is copied to DUMP_DIRECTORY/podName.
//
// namespace - The namespace of the target pod
// podName - The name of the pod
// containerName - The target container name within the pod
// containerLogsDir - The logs directory location within the container
//
// DUMP_KUBECONFIG - The kube config file to use when executing the cluster dump tool.
// DUMP_DIRECTORY - The directory to store the cluster dump within.
func DumpContainerLogs(namespace string, podName string, containerName string, containerLogsDir string) {
	directory := os.Getenv(test.DumpDirectoryEnvVarName)
	kubeconfig := os.Getenv(test.DumpKubeconfigEnvVarName)

	containerPath := fmt.Sprintf("%s/%s:%s", namespace, podName, containerLogsDir)
	destDir := fmt.Sprintf("%s/%s/%s", directory, podName, containerName)

	cmd := exec.Command("kubectl", "cp", containerPath, "-c", containerName, destDir)
	Log(Info, fmt.Sprintf("kubectl command to dump %s logs: %s", podName, cmd.String()))
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
