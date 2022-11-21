// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package files

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

var vpoLog = "../../../test/cluster/ingress-ip-not-found/cluster-snapshot/verrazzano-install/verrazzano-platform-wls-64694f7cc4-br684/logs.txt"
var ingressError = "Failed getting DNS suffix: No IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer"

func TestFilterInstallLog(t *testing.T) {
	allMessages, _ := ConvertToLogMessage(vpoLog)
	vmoErrors, _ := FilterLogsByLevelComponent("error", "verrazzano-monitoring-wls", allMessages)
	assert.True(t, len(vmoErrors) > 0)
	errorMessage := vmoErrors[len(vmoErrors)-1].Message
	assert.True(t, errorMessage == ingressError)
}

func TestWrongInstallLog(t *testing.T) {
	vpoLog = "../../../test/cluster/ingress-ip-not-found/cluster-snapshot/verrazzano-install-wrong/verrazzano-platform-wls-64694f7cc4-br684/logs.txt"
	_, err := ConvertToLogMessage(vpoLog)
	errorMessage := vpoLog + ": no such file or directory"
	assert.True(t, strings.Contains(err.Error(), errorMessage))
}
