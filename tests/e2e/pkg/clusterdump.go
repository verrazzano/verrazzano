// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"fmt"
	"os"
	"os/exec"
)

// ExecuteAnalysis executes the verrazzano-analysis tool.
// command - The fully qualified verrazzano-analysis executable.
// directory - The directory containing the cluster dump to analyze.
func ExecuteAnalysis(analysisCommand string, directory string) error {
	fmt.Printf("Execute analysis: %s --reportFile %s/analysis.report %s\n", analysisCommand, directory, directory)
	if analysisCommand == "" {
		return nil
	}
	reportFile := fmt.Sprintf("%s/analysis.report", directory)
	cmd := exec.Command(analysisCommand, "--reportFile ", reportFile, directory)
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

// ExecuteClusterDump executes the cluster dump tool.
// command - The fully qualified cluster dump executable.
// kubeconfig - The kube config file to use when executing the cluster dump tool.
// directory - The directory to store the cluster dump within.
func ExecuteClusterDump(dumpCommand string, analysisCommand string, kubeconfig string, directory string) error {
	fmt.Printf("Execute cluster dump: KUBECONFIG=%s; %s -d %s\n", kubeconfig, dumpCommand, directory)
	if dumpCommand == "" {
		return nil
	}
	cmd := exec.Command(dumpCommand, "-d", directory)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
        err = ExecuteAnalysis(analysisCommand, directory)

	return nil
}

// ExecuteClusterDumpWithEnvVarConfig executes the cluster dump tool using config from environment variables.
// DUMP_KUBECONFIG - The kube config file to use when executing the cluster dump tool.
// DUMP_DIRECTORY - The directory to store the cluster dump within.
// DUMP_COMMAND - The fully quallified cluster dump executable.
func ExecuteClusterDumpWithEnvVarConfig() error {
	kubeconfig := os.Getenv("DUMP_KUBECONFIG")
	directory := os.Getenv("DUMP_DIRECTORY")
	dumpCommand := os.Getenv("DUMP_COMMAND")
	analysisCommand := os.Getenv("ANALYSIS_COMMAND")
	return ExecuteClusterDump(dumpCommand, analysisCommand, kubeconfig, directory)
}
