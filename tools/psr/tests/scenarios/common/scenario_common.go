// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/onsi/gomega"
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
	_, err := pkg.CreateOrUpdateNamespace(namespace, constants.NamespaceLabels, nil)
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	err = secrets.CreateOrUpdatePipelineImagePullSecret(log, namespace, kubeconfig)
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))

	if skipStartScenario {
		return
	}

	if !psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		stdout, stderr, err := psrctlcli.StartScenario(log, scenarioID, namespace)
		t.Logs.Infof("StartScenario %s/%s, stdout: %s, stderr: %s", namespace, scenarioID, stdout, stderr)
		gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	}
}

// StopScenario - Stops a scenario if it is running in the specified namespace; if skipStopScenario is true, we don't shut it down
func StopScenario(t *framework.TestFramework, log vzlog.VerrazzanoLogger, scenarioID string, namespace string, skipStopScenario bool) {
	if skipStopScenario {
		t.Logs.Info("Skip stop scenario")
		return
	}
	if psrctlcli.IsScenarioRunning(log, scenarioID, namespace) {
		stdout, stderr, err := psrctlcli.StopScenario(log, scenarioID, namespace)
		t.Logs.Infof("StopScenario %s/%s, stdout: %s, stderr: %s", namespace, scenarioID, stdout, stderr)
		gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
	}
}
