// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestUpdateOperatorYAMLPrivateRegistry(t *testing.T) {
	myReg := "fra.ocir.io"
	myPrefix := "mypref"
	fname, err := updateOperatorYAMLPrivateRegistry("../../test/testdata/operator-file-fake.yaml", myReg, myPrefix)
	assert.NoError(t, err)
	fmt.Printf("finished processing file, wrote %s\n", fname)
	editedOperatorFile, err := os.Open(fname)
	assert.NoError(t, err)
	defer func() { editedOperatorFile.Close() }()
	opYAML, err := k8sutil.Unmarshall(bufio.NewReader(editedOperatorFile))
	assert.NoError(t, err)
	assert.NotNil(t, opYAML)
	vpoIdx, webhookIdx := findVPODeploymentIndices(opYAML)
	assert.NotEqual(t, -1, vpoIdx)
	assert.NotEqual(t, -1, webhookIdx)
	vpoDeploy := opYAML[vpoIdx]
	webhookDeploy := opYAML[webhookIdx]
	expectedImagePrefix := fmt.Sprintf("%s/%s/verrazzano/verrazzano-platform-operator:", myReg, myPrefix)
	assertAllPrivateRegImageNames(t, vpoDeploy, expectedImagePrefix)
	assertAllPrivateRegImageNames(t, webhookDeploy, expectedImagePrefix)
	assertPrivateRegEnvVars(t, vpoDeploy, myReg, myPrefix)
}

func assertPrivateRegEnvVars(t *testing.T, deployment unstructured.Unstructured, imageRegistry string, imagePrefix string) {
	containers, _, err := unstructured.NestedSlice(deployment.Object, containersFields()...)
	assert.NoError(t, err)
	registryOk := false
	prefixOk := false
	for _, ctr := range containers {
		container := ctr.(map[string]interface{})
		if container["name"] == constants.VerrazzanoPlatformOperator {
			env := container["env"].([]interface{})
			for _, eachEnv := range env {
				envVar := eachEnv.(map[string]interface{})
				varName := envVar["name"]
				if varName == vpoconst.RegistryOverrideEnvVar {
					val := envVar["value"]
					assert.Equal(t, imageRegistry, val)
					registryOk = true
					continue
				}
				if varName == vpoconst.ImageRepoOverrideEnvVar {
					val := envVar["value"]
					assert.Equal(t, imagePrefix, val)
					prefixOk = true
				}
			}
		}
	}

	assert.True(t, registryOk && prefixOk, "Expected both registry and prefix env vars to be populated")
}

func assertAllPrivateRegImageNames(t *testing.T, deployment unstructured.Unstructured, expectedImagePrefix string) {
	containers, _, err := unstructured.NestedSlice(deployment.Object, containersFields()...)
	assert.NoError(t, err)
	assertImageNamesStartWith(t, expectedImagePrefix, containers)
	initContainers, _, err := unstructured.NestedSlice(deployment.Object, initContainersFields()...)
	assert.NoError(t, err)
	assertImageNamesStartWith(t, expectedImagePrefix, initContainers)
}

func assertImageNamesStartWith(t *testing.T, expectedImagePrefix string, containers []interface{}) {
	for _, ctr := range containers {
		container := ctr.(map[string]interface{})
		assert.Truef(t, strings.HasPrefix(container["image"].(string), expectedImagePrefix),
			"Expected container %s to have image name starting with %s, but found %s",
			container["name"], expectedImagePrefix, container["image"])
	}
}
