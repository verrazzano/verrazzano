// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package explain

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	corev1cli "k8s.io/client-go/kubernetes/typed/core/v1"
)

const psrRoot = "../../.."

var (
	expectedID          = "ops-s1"
	expectedName        = "opensearch-s1"
	expectedDescription = "This is a scenario that writes logs to STDOUT and gets logs from OpenSearch at a moderated rate."
	expectedUseCase     = "Usecase path opensearch/writelogs.yaml:  Description: write logs to STDOUT 10 times a second"
)

// TestExplainScenario tests the NewCmdExplain and RunCmdExplain functions
//
//	WHEN 'psrctl explain -s ops-s1' is called
//	THEN ensure the output is correct for that scenario
func TestExplainScenario(t *testing.T) {
	manifest.Manifests = &manifest.PsrManifests{
		RootTmpDir:        psrRoot,
		WorkerChartAbsDir: psrRoot + "/manifests/charts/worker",
		UseCasesAbsDir:    psrRoot + "/manifests/usecases",
		ScenarioAbsDir:    psrRoot + "/manifests/scenarios",
	}

	defer manifest.ResetManifests()

	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	k8sutil.GetCoreV1Func = func(log ...vzlog.VerrazzanoLogger) (corev1cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	explainCmd := NewCmdExplain(rc)
	explainCmd.PersistentFlags().Set(constants.FlagScenario, "ops-s1")
	explainCmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	assert.NotNil(t, explainCmd)

	// Run explain command, check for the expected status results to be displayed
	err := explainCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	assert.Contains(t, result, fmt.Sprintf("ID: %s", expectedID))
	assert.Contains(t, result, fmt.Sprintf("Name: %s", expectedName))
	assert.Contains(t, result, fmt.Sprintf("Description: %s", expectedDescription))
	assert.NotContains(t, result, "ID: ops-s9")
	assert.NotContains(t, result, fmt.Sprintf("Use cases: %s", expectedUseCase))
}

// TestExplainScenarioVerbose tests the NewCmdExplain and RunCmdExplain functions
//
//	WHEN 'psrctl explain -s ops-s1 -v' is called
//	THEN ensure the output includes the verbose usecases
func TestExplainScenarioVerbose(t *testing.T) {
	manifest.Manifests = &manifest.PsrManifests{
		RootTmpDir:        psrRoot,
		WorkerChartAbsDir: psrRoot + "/manifests/charts/worker",
		UseCasesAbsDir:    psrRoot + "/manifests/usecases",
		ScenarioAbsDir:    psrRoot + "/manifests/scenarios",
	}

	defer manifest.ResetManifests()

	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	k8sutil.GetCoreV1Func = func(log ...vzlog.VerrazzanoLogger) (corev1cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	explainCmd := NewCmdExplain(rc)
	explainCmd.PersistentFlags().Set(constants.FlagScenario, "ops-s1")
	explainCmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	explainCmd.PersistentFlags().Set(constants.FlagVerbose, "true")
	assert.NotNil(t, explainCmd)

	// Run explain command, check for the expected status results to be displayed
	err := explainCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	assert.Contains(t, result, fmt.Sprintf("ID: %s", expectedID))
	assert.Contains(t, result, fmt.Sprintf("Name: %s", expectedName))
	assert.Contains(t, result, fmt.Sprintf("Description: %s", expectedDescription))
	assert.Contains(t, result, fmt.Sprintf("Use cases:\n%s", expectedUseCase))
	assert.NotContains(t, result, "ID: ops-s9")
}

// TestExplainNoScenario tests the NewCmdExplain and RunCmdExplain functions
//
//	WHEN 'psrctl explain' is called
//	THEN ensure the output correctly lists all scenarios
func TestExplainNoScenario(t *testing.T) {
	manifest.Manifests = &manifest.PsrManifests{
		RootTmpDir:        psrRoot,
		WorkerChartAbsDir: psrRoot + "/manifests/charts/worker",
		UseCasesAbsDir:    psrRoot + "/manifests/usecases",
		ScenarioAbsDir:    psrRoot + "/manifests/scenarios",
	}

	defer manifest.ResetManifests()

	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	k8sutil.GetCoreV1Func = func(log ...vzlog.VerrazzanoLogger) (corev1cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	explainCmd := NewCmdExplain(rc)
	assert.NotNil(t, explainCmd)

	// Run explain command, check for all scenarios to be listed
	err := explainCmd.Execute()
	assert.NoError(t, err)
	result := buf.String()
	assert.Contains(t, result, "ID: ops-s9")
	assert.Contains(t, result, "Name: opensearch-s9")
	assert.Contains(t, result, "Description: This is a scenario that combines all of the existing OpenSearch use cases")
	assert.Contains(t, result, "Namespace needs to be labeled with istio-injection=enabled")
	assert.NotContains(t, result, fmt.Sprintf("Use cases: %s", expectedUseCase))
}
