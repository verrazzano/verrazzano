// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package start

import (
	"bytes"
	"encoding/base64"
	"fmt"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	corev1cli "k8s.io/client-go/kubernetes/typed/core/v1"
)

const psrRoot = "../../.."

var ID = "ops-s1"

// TestStartCmd tests the NewCmdStart and RunCmdStart functions
//
//	WHEN 'psrctl start -s ops-s1 -n psr' is called
//	THEN ensure the new scenario gets started
func TestStartCmd(t *testing.T) {
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

	defer func() { scenario.StartUpgradeFunc = helmcli.Upgrade }()
	scenario.StartUpgradeFunc = func(log vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		//assert.Equal(t, 3, len(overrides))
		//assert.Equal(t, "psr-ops-s1-writelogs-0", releaseName)
		//assert.Equal(t, "psr", namespace)
		//assert.Contains(t, chartDir, "manifests/charts/worker")

		// TODO: add start assertions

		return nil, nil, nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	cmd := NewCmdStart(rc)
	cmd.PersistentFlags().Set(constants.FlagScenario, "ops-s1")
	cmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	assert.NotNil(t, cmd)

	// Run explain command, check for the expected status results to be displayed
	err := cmd.Execute()
	assert.NoError(t, err)
	result := buf.String()

	assert.Contains(t, result, fmt.Sprintf("Starting scenario %s", ID))
	assert.Contains(t, result, fmt.Sprintf("Scenario %s successfully started", ID))
}

// TestStartExisting tests the NewCmdStart and RunCmdStart functions
//
//	WHEN 'psrctl start -s ops-s1 -n psr' is called when the scenario is already running
//	THEN ensure the output correctly tells the user their operation is invalid
func TestStartExisting(t *testing.T) {
	manifest.Manifests = &manifest.PsrManifests{
		RootTmpDir:        psrRoot,
		WorkerChartAbsDir: psrRoot + "/manifests/charts/worker",
		UseCasesAbsDir:    psrRoot + "/manifests/usecases",
		ScenarioAbsDir:    psrRoot + "/manifests/scenarios",
	}

	// create scenario ConfigMap
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "psr-ops-s1",
			Namespace: "psr",
			Labels: map[string]string{
				"psr.verrazzano.io/scenario":    "true",
				"psr.verrazzano.io/scenario-id": "ops-s1",
			},
		},
		Data: map[string]string{
			"scenario": base64.StdEncoding.EncodeToString([]byte(`Description: "This is a scenario that writes logs to STDOUT and gets logs from OpenSearch
  at a moderated rate. \nThe purpose of the scenario is to test a moderate load on
  both Fluend and OpenSearch by logging records.\n"
HelmReleases:
- Description: write logs to STDOUT 10 times a second
  Name: psr-ops-s1-writelogs-0
  Namespace: psr
  OverrideFile: writelogs.yaml
  UsecasePath: opensearch/writelogs.yaml
ID: ops-s1
Name: opensearch-s1
Namespace: default
ScenarioUsecaseOverridesAbsDir: temp-dir
Usecases:
- Description: write logs to STDOUT 10 times a second
  OverrideFile: writelogs.yaml
  UsecasePath: opensearch/writelogs.yaml
`)),
		},
	}

	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	k8sutil.GetCoreV1Func = func(log ...vzlog.VerrazzanoLogger) (corev1cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset(cm).CoreV1(), nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	cmd := NewCmdStart(rc)
	cmd.PersistentFlags().Set(constants.FlagScenario, "ops-s1")
	cmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	assert.NotNil(t, cmd)

	// Run start command, check for the expected error to be thrown
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("Scenario %s already running in namespace psr", ID))

	result := buf.String()
	assert.Contains(t, result, fmt.Sprintf("Starting scenario %s", ID))
}

// TestStartInvalid tests the NewCmdStart and RunCmdStart functions
//
//	WHEN 'psrctl start -s bad-manifest -n psr' is called when the scenario does not exist
//	THEN ensure the output correctly tells the user their operation is invalid
func TestStartInvalid(t *testing.T) {

	manifest.Manifests = &manifest.PsrManifests{
		RootTmpDir:        psrRoot,
		WorkerChartAbsDir: psrRoot + "/manifests/charts/worker",
		UseCasesAbsDir:    psrRoot + "/manifests/usecases",
		ScenarioAbsDir:    psrRoot + "/manifests/scenarios",
	}

	defer func() { k8sutil.GetCoreV1Func = k8sutil.GetCoreV1Client }()
	k8sutil.GetCoreV1Func = func(log ...vzlog.VerrazzanoLogger) (corev1cli.CoreV1Interface, error) {
		return k8sfake.NewSimpleClientset().CoreV1(), nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	cmd := NewCmdStart(rc)
	cmd.PersistentFlags().Set(constants.FlagScenario, "bad-manifest")
	cmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	assert.NotNil(t, cmd)

	// Run explain command, check for the expected status results to be displayed
	err := cmd.Execute()
	assert.Error(t, err)
	result := err.Error()

	assert.Contains(t, result, "Failed to find scenario manifest with ID bad-manifest")
}
