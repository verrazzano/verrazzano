package explain

import (
	"bytes"
	"encoding/base64"
	"fmt"
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

var (
	expectedName        = "opensearch-s1"
	expectedId          = "ops-s1"
	expectedDescription = "This is a scenario that writes logs to STDOUT and gets logs from OpenSearch at a moderated rate."
	// ConfigMap for the ops-s1 scenario
	cm = &corev1.ConfigMap{
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
)

// TestExplainScenario tests the NewCmdExplain and RunCmdExplain functions
//
//	WHEN 'psr explain -s ops-s1' is called
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
		return k8sfake.NewSimpleClientset(cm).CoreV1(), nil
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
	assert.Contains(t, fmt.Sprintf("Name: %s", expectedName), result)
	assert.Contains(t, fmt.Sprintf("ID: %s", expectedId), result)
	assert.Contains(t, fmt.Sprintf("Description: %s", expectedDescription), result)
}

// TestExplainScenarioVerbose tests the NewCmdExplain and RunCmdExplain functions
//
//	WHEN 'psr explain -s ops-s2 -v' is called
//	THEN ensure the output is correct for that scenario
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
		return k8sfake.NewSimpleClientset(cm).CoreV1(), nil
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
	assert.Contains(t, fmt.Sprintf("Name: %s", expectedName), result)
	assert.Contains(t, fmt.Sprintf("ID: %s", expectedId), result)
	assert.Contains(t, fmt.Sprintf("Description: %s", expectedDescription), result)
}

// TestExplainNoScenario tests the NewCmdExplain and RunCmdExplain functions
//
//	WHEN 'psr explain' is called
//	THEN ensure the output is correct for all listed scenarios
func TestExplainNoScenario(t *testing.T) {

}
