// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/psrctlcli"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/secrets"
)

// InitScenario Starts a PSR scenario in the specified namespace; if skipStartScenario is true, the scenario start is skipped
// - Creates and labels and the namespace if necessary
// - If the env var IMAGE_PULL_SECRET is defined, we attempt to create the image pull secret in the target namespace
func InitScenario(t *framework.TestFramework, log vzlog.VerrazzanoLogger, scenarioID string, namespace string, kubeconfig string, skipStartScenario bool) {
	if _, err := pkg.CreateOrUpdateNamespace(namespace, constants.NamespaceLabels, nil); err != nil {
		t.Fail(fmt.Sprintf("Error creating or updating namespace %s: %s", namespace, err.Error()))
		return
	}

	if err := secrets.CreateOrUpdatePipelineImagePullSecret(log, namespace, kubeconfig); err != nil {
		t.Fail(fmt.Sprintf("Error creating creating image pull secret for tests suite: %s", err.Error()))
		return
	}

	if skipStartScenario {
		return
	}

	if !psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		_, stderr, err := psrctlcli.StartScenario(log, scenarioID, namespace)
		if err != nil {
			t.Fail(fmt.Sprintf("Error starting scenario: %s", err.Error()))
			log.Error(string(stderr))
			return
		}
	}
}

// StopScenario - Stops a scenario if it is running in the specified namespace; if skipStopScenario is true, we don't shut it down
func StopScenario(t *framework.TestFramework, log vzlog.VerrazzanoLogger, scenarioID string, namespace string, skipStopScenario bool) {
	if !skipStopScenario && psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		_, stderr, err := psrctlcli.StopScenario(log, scenarioID, namespace)
		if err != nil {
			log.Errorf("Error starting scenario: %s", err.Error())
			log.Error(string(stderr))
		}
	}
}
