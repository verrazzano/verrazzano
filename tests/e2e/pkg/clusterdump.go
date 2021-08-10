// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"os"
	"os/exec"
)

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

// ExecuteClusterDumpWithEnvVarConfig executes the cluster dump tool using config from environment variables.
// DUMP_KUBECONFIG - The kube config file to use when executing the cluster dump tool.
// DUMP_DIRECTORY - The directory to store the cluster dump within.
// DUMP_COMMAND - The fully quallified cluster dump executable.
func ExecuteClusterDumpWithEnvVarConfig() error {
	kubeconfig := os.Getenv("DUMP_KUBECONFIG")
	directory := os.Getenv("DUMP_DIRECTORY")
	command := os.Getenv("DUMP_COMMAND")
	return ExecuteClusterDump(command, kubeconfig, directory)
}
