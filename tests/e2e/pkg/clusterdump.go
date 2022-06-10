// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test"
	"os"
	"os/exec"
	"path/filepath"
	"time"
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
		}

		// ginkgo.Fail and gomega matchers panic if they fail. Recover is used to capture the panic and
		// generate the cluster dump
		defer func() {
			if r := recover(); r != nil {
				ExecuteClusterDumpWithEnvVarSuffix(fmt.Sprintf("aftersuite-%d", time.Now().Unix()))
			}
		}()
		body()
	})
}

// ExecuteClusterDump executes the cluster dump tool.
// command - The fully qualified cluster dump executable.
// kubeconfig - The kube config file to use when executing the cluster dump tool.
// directory - The directory to store the cluster dump within.
func ExecuteClusterDump(command string, kubeconfig string, directory string) error {
	fmt.Printf("Execute cluster dump: KUBECONFIG=%s; %s -d %s\n", kubeconfig, command, directory)
	if command == "" {
		return nil
	}
	reportFile := fmt.Sprintf("%s/cluster-dump/analysis.report", directory)
	cmd := exec.Command(command, "-d", directory, "-r", reportFile)
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

func ExecuteClusterDumpWithEnvVarSuffix(directorySuffix string) error {
	kubeconfig := os.Getenv("DUMP_KUBECONFIG")
	directory := filepath.Join(os.Getenv("DUMP_DIRECTORY"), directorySuffix)
	command := os.Getenv("DUMP_COMMAND")
	return ExecuteClusterDump(command, kubeconfig, directory)
}

// ExecuteClusterDumpWithEnvVarConfig executes the cluster dump tool using config from environment variables.
// DUMP_KUBECONFIG - The kube config file to use when executing the cluster dump tool.
// DUMP_DIRECTORY - The directory to store the cluster dump within.
// DUMP_COMMAND - The fully quallified cluster dump executable.
func ExecuteClusterDumpWithEnvVarConfig() error {
	return ExecuteClusterDumpWithEnvVarSuffix("")
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
