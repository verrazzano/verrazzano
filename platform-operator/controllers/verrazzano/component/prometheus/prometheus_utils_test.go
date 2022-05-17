// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
)

func TestGetVerrazzanoMonitoringNamespace(t *testing.T) {
	ns := GetVerrazzanoMonitoringNamespace()
	assert.Equal(t, "enabled", ns.Labels[v8oconst.LabelIstioInjection])
	assert.Equal(t, vpoconst.VerrazzanoMonitoringNamespace, ns.Labels[v8oconst.LabelVerrazzanoNamespace])
	assert.Equal(t, vpoconst.VerrazzanoMonitoringNamespace, ns.Name)
}
