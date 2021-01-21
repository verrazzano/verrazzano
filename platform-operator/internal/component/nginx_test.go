// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"os/exec"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
)

// nginxFakeRunner is used to test helm without actually running an OS exec command
type nginxFakeRunner struct {
}

// TestNginxName tests the ingress-nginx component name
// GIVEN a ingress-nginx component
//  WHEN I get the Name
//  THEN the ingress-nginx component name is returned
func TestNginxName(t *testing.T) {
	assert := assert.New(t)
	comp := Nginx{}
	assert.Equal("ingress-nginx", comp.Name(), "Wrong component name for ingress-nginx")
}

// TestNginxUpgrade tests the ingress-nginx component upgrade
// GIVEN a ingress-nginx component
//  WHEN I call Upgrade
//  THEN the ingress-nginx upgrade returns success
func TestNginxUpgrade(t *testing.T) {
	const defNs = "ingress-nginx"
	assert := assert.New(t)
	comp := Nginx{}
	helm.SetCmdRunner(nginxFakeRunner{})
	err := comp.Upgrade(nil, "")
	assert.NoError(err, "Upgrade returned an error")
}

// TestNginxResolveNamespace tests the ingress-nginx component namespace
// GIVEN a ingress-nginx component
//  WHEN I call resolveNamespace
//  THEN the ingress-nginx namespace name is correctly resolved
func TestNginxResolveNamespace(t *testing.T) {
	const defNs = "ingress-nginx"
	assert := assert.New(t)
	ns := nginxNamespace("")
	assert.Equal(defNs, ns, "Wrong namespace resolved for ingress-nginx when using empty namespace")
	ns = nginxNamespace("default")
	assert.Equal(defNs, ns, "Wrong namespace resolved for ingress-nginx when using default namespace")
	ns = nginxNamespace("custom")
	assert.Equal("custom", ns, "Wrong namespace resolved for ingress-nginx when using custom namesapce")
}

func (r nginxFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
