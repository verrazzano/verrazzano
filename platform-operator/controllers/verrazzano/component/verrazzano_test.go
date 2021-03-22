// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
	"go.uber.org/zap"
)

// fakeRunner is used to test helm without actually running an OS exec command
type fakeRunner struct {
}

// TestVzName tests the Verrazzano component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the Verrazzano component name is returned
func TestVzName(t *testing.T) {
	assert := assert.New(t)
	vz := Verrazzano{}
	assert.Equal("verrazzano", vz.Name(), "Wrong component name for verrazzano")
}

// TestVzUpgrade tests the Verrazzano component name
// GIVEN a Verrazzano component
//  WHEN I call Upgrade
//  THEN the Verrazzano upgrade returns success
func TestVzUpgrade(t *testing.T) {
	assert := assert.New(t)
	vz := Verrazzano{}
	helm.SetCmdRunner(fakeRunner{})
	defer helm.SetDefaultRunner()
	err := vz.Upgrade(zap.S(), nil, "")
	assert.NoError(err, "Upgrade returned an error")
}

// TestVzResolveNamespace tests the Verrazzano component name
// GIVEN a Verrazzano component
//  WHEN I call resolveNamespace
//  THEN the Verrazzano namespace name is correctly resolved
func TestVzResolveNamespace(t *testing.T) {
	const defNs = constants.VerrazzanoSystemNamespace
	assert := assert.New(t)
	ns := resolveNamespace("")
	assert.Equal(defNs, ns, "Wrong namespace resolved for verrazzano when using empty namespace")
	ns = resolveNamespace("default")
	assert.Equal(defNs, ns, "Wrong namespace resolved for verrazzano when using default namespace")
	ns = resolveNamespace("custom")
	assert.Equal("custom", ns, "Wrong namespace resolved for verrazzano when using custom namesapce")
}

func (r fakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
