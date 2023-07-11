// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAuthProxyPrincipal(t *testing.T) {
	asserts := assert.New(t)
	asserts.Equal("cluster.local/ns/verrazzano-system/sa/verrazzano-authproxy", GetAuthProxyPrincipal())
}

func TestGetGrafanaPrincipal(t *testing.T) {
	asserts := assert.New(t)
	asserts.Equal("cluster.local/ns/verrazzano-system/sa/verrazzano-monitoring-operator", GetVMOPrincipal())
}

func TestGetKialiPrincipal(t *testing.T) {
	asserts := assert.New(t)
	asserts.Equal("cluster.local/ns/verrazzano-system/sa/vmi-system-kiali", GetKialiPrincipal())
}

func TestGetJaegerPrincipal(t *testing.T) {
	asserts := assert.New(t)
	asserts.Equal("cluster.local/ns/verrazzano-monitoring/sa/jaeger-operator-jaeger", GetJaegerPrincipal())
}

func TestGetThanosQueryPrincipal(t *testing.T) {
	asserts := assert.New(t)
	asserts.Equal("cluster.local/ns/verrazzano-monitoring/sa/thanos-query", GetThanosQueryPrincipal())
}
