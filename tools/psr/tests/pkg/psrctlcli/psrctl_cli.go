// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package psrctlcli

import (
	"encoding/json"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	"os"
	"os/exec"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
)

// Debug is set from a platform-operator arg and sets the helm --debug flag
var Debug bool

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

const PsrctlCmdKey = "PSR_COMMAND"

// GetPsrctlCmd Returns the path to the psrctl command for the test run
func GetPsrctlCmd() string {
	psrcmd := os.Getenv(PsrctlCmdKey)
	if psrcmd == "" {
		psrcmd = "psrctl"
	}
	return psrcmd
}

// IsScenarioRunning Returns true if the named scenario is running
func IsScenarioRunning(log vzlog.VerrazzanoLogger, scenarioName string, namespace string) bool {
	_, found := FindScenario(log, scenarioName, namespace)
	return found
}

// FindScenario Returns true if the named scenario is running
func FindScenario(log vzlog.VerrazzanoLogger, scenarioID string, namespace string) (scenario.Scenario, bool) {
	runningScenarios, err := ListScenarios(log, namespace)
	if err != nil {
		log.Errorf("Error listing scenarios: %s", err.Error())
		return scenario.Scenario{}, false
	}
	for _, scenario := range runningScenarios {
		if scenario.ScenarioManifest.ID == scenarioID {
			return scenario, true
		}
	}
	return scenario.Scenario{}, false
}

// ListScenarios Lists any running scenarios in the specified namespace
func ListScenarios(log vzlog.VerrazzanoLogger, namespace string) ([]scenario.Scenario, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"list", "-o", "json"}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	psrctlCmd := GetPsrctlCmd()
	log.Infof("psrctl: %s", psrctlCmd)
	cmd := exec.Command(psrctlCmd, args...)
	log.Debugf("Running command to list scenarios: %s", cmd.String())
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		log.Errorf("Failed to list scenarios for namespace %s: stderr %s", namespace, string(stderr))
		return []scenario.Scenario{}, err
	}

	//  Log get values output
	log.Debugf("Successfully listed scenarios in namespace %s: %v", namespace, string(stdout))
	var scenarios []scenario.Scenario
	if err := json.Unmarshal(stdout, &scenarios); err != nil {
		return nil, err
	}
	return scenarios, nil
}

// StartScenario Starts a PSR scenario in the specified namespace.
func StartScenario(log vzlog.VerrazzanoLogger, scenario string, namespace string, additionalArgs ...string) ([]byte, []byte, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"start", "-s", scenario, "-n", namespace}
	args = append(args, additionalArgs...)

	psrctlCmd := GetPsrctlCmd()
	cmd := exec.Command(psrctlCmd, args...)
	log.Debugf("Run scenario command: %s", cmd.String())
	return runner.Run(cmd)
}

// StopScenario Starts a PSR scenario in the specified namespace.
func StopScenario(log vzlog.VerrazzanoLogger, scenario string, namespace string, additionalArgs ...string) ([]byte, []byte, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"stop", "-s", scenario, "-n", namespace}
	args = append(args, additionalArgs...)

	psrctlCmd := GetPsrctlCmd()
	cmd := exec.Command(psrctlCmd, args...)
	log.Debugf("Run scenario command: %s", cmd.String())
	return runner.Run(cmd)
}
