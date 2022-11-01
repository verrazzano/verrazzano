// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"github.com/stretchr/testify/assert"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"os"
	"testing"
)

func Test(t *testing.T) {

	chartDir, err := unpackWorkerChartToDir()
	assert.NoError(t, err)
	helmcli.Upgrade(vzlog.DefaultLogger(), "psrcli", "default", chartDir, true, false, nil)
	os.RemoveAll(chartDir)

}
