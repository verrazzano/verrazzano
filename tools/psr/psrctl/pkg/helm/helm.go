// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"os"
)

func InstallScenario() (string, error) {
	chartDir, err := embedded.CreateTempChartDir()
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(chartDir)

	err = embedded.CopyWorkerChartToTempDir(chartDir)
	if err != nil {
		return "", err
	}

	var stdout, stderr []byte
	stdout, stderr, err = helmcli.Upgrade(vzlog.DefaultLogger(), "psrcli", "default", chartDir, true, false, nil)
	if err != nil {
		return string(stderr), err
	}
	return string(stdout), err
}
