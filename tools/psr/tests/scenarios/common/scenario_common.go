// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/psrctlcli"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/secrets"
	"strings"
)

var (
	NamespaceLabels = map[string]string{
		"istio-injection":    "enabled",
		"verrazzano-managed": "true",
	}
)

// InitScenario Starts a PSR scenario in the specified namespace; if skipStartScenario is true, the scenario start is skipped
// - Creates and labels and the namespace if necessary
// - If the env var IMAGE_PULL_SECRET is defined, we attempt to create the image pull secret in the target namespace
func InitScenario(t *framework.TestFramework, log vzlog.VerrazzanoLogger, scenarioID string, namespace string, kubeconfig string, skipStartScenario bool) {
	_, err := pkg.CreateOrUpdateNamespace(namespace, NamespaceLabels, nil)
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	err = secrets.CreateOrUpdatePipelineImagePullSecret(log, namespace, kubeconfig)
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	if skipStartScenario {
		return
	}

	if psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		t.Logs.Infof("Scenario %s/%s is already running", namespace, scenarioID)
		return
	}
	stdout, stderr, err := psrctlcli.StartScenario(log, scenarioID, namespace)
	if err != nil {
		// When running in parallel, the command may fail due to a race condition
		if !strings.Contains(string(stderr), "release: already exists") {
			t.Fail(fmt.Sprintf("Unexpected error starting scenario: %s", string(stderr)))
			return
		}
	}
	t.Logs.Infof("StartScenario %s/%s successful, stdout: %s, stderr: %s", namespace, scenarioID, stdout, stderr)
}

// StopScenario - Stops a scenario if it is running in the specified namespace; if skipStopScenario is true, we don't shut it down
func StopScenario(t *framework.TestFramework, log vzlog.VerrazzanoLogger, scenarioID string, namespace string, skipStopScenario bool) {
	if skipStopScenario {
		t.Logs.Info("Skip stop scenario")
		return
	}
	if !psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		t.Logs.Infof("Scenario %s/%s is not running", namespace, scenarioID)
		return
	}
	stdout, stderr, err := psrctlcli.StopScenario(log, scenarioID, namespace)
	if err != nil {
		stderrString := string(stderr)
		if !strings.Contains(stderrString, "Failed to purge the release: release: not found") &&
			!strings.Contains(stderrString, "Failed to find ConfigMap for scenario") &&
			!strings.Contains(stderrString, "Failed to purge the release: secrets") {
			t.Fail(fmt.Sprintf("Unexpected error starting scenario: %s", stderrString))
			return
		}
	}
	t.Logs.Infof("StopScenario %s/%s succeeded, stdout: %s, stderr: %s", namespace, scenarioID, stdout, stderr)
}
