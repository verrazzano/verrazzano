// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"github.com/onsi/ginkgo/v2"

	"testing"
)

func TestGrafanaDashboardPreUpgrade(test *testing.T) {
	t.RegisterFailHandler()
	ginkgo.RunSpecs(test, "Pre Upgrade Grafana Dashboard Suite")
}
