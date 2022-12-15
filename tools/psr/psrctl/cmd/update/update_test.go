// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	"bytes"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/scenario"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	corev1cli "k8s.io/client-go/kubernetes/typed/core/v1"
)

const psrRoot = "../../.."

// TestUpdateCmd tests that the psrctl command works correctly
func TestUpdateCmd(t *testing.T) {
	manifest.Manifests = &manifest.PsrManifests{
		RootTmpDir:        psrRoot,
		WorkerChartAbsDir: psrRoot + "/manifests/charts/worker",
		UseCasesAbsDir:    psrRoot + "/manifests/usecases",
		ScenarioAbsDir:    psrRoot + "/manifests/scenarios",
	}

	defer manifest.ResetManifests()

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

	defer func() { scenario.UpdateGetValuesFunc = helmcli.GetValues }()
	scenario.UpdateGetValuesFunc = func(log vzlog.VerrazzanoLogger, releaseName string, namespace string) ([]byte, error) {
		assert.Equal(t, "psr-ops-s1-writelogs-0", releaseName)
		assert.Equal(t, "psr", namespace)
		return []byte("old-values"), nil
	}

	defer func() { scenario.UpdateUpgradeFunc = helmcli.Upgrade }()
	scenario.UpdateUpgradeFunc = func(log vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helmcli.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		assert.Equal(t, 3, len(overrides))
		assert.Equal(t, "psr-ops-s1-writelogs-0", releaseName)
		assert.Equal(t, "psr", namespace)
		assert.Contains(t, chartDir, "manifests/charts/worker")
		return nil, nil, nil
	}

	// Send the command output to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})

	cmd := NewCmdUpdate(rc)
	assert.NotNil(t, cmd)

	cmd.PersistentFlags().Set(constants.FlagScenario, "ops-s1")
	cmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	cmd.PersistentFlags().Set(constants.ImageNameKey, "worker-image")

	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Updating scenario")
	assert.Contains(t, buf.String(), "Updating use case")
	assert.Contains(t, buf.String(), "successfully updated")
}

func TestUpdateNoConfigmap(t *testing.T) {
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

	cmd := NewCmdUpdate(rc)
	assert.NotNil(t, cmd)

	cmd.PersistentFlags().Set(constants.FlagScenario, "ops-s1")
	cmd.PersistentFlags().Set(constants.FlagNamespace, "psr")
	cmd.PersistentFlags().Set(constants.ImageNameKey, "worker-image")

	err := cmd.Execute()
	assert.Error(t, err)
}
