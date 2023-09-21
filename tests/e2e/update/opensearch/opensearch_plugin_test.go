// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearch

import (
	. "github.com/onsi/ginkgo/v2"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

var _ = t.Describe("Update Plugins", Label("f:platform-plugin.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN opensearch plugin is updated with wrong value
	// THEN master pods gets in failure state.
	// In last opensearch plugin is disabled
	// Then all pods are back to normal state
	t.It("opensearch update plugin", func() {
		m := OpenSearchPlugins{Enabled: true, InstanceList: "abc"}
		update.UpdateCRWithPlugins(m, pollingInterval, waitTimeout)
		update.ValidatePods(string(vmov1.MasterRole), NodeGroupLabel, constants.VerrazzanoSystemNamespace, 0, false)
		m = OpenSearchPlugins{Enabled: false, InstanceList: "analysis-stempel"}
		update.UpdateCRWithPlugins(m, pollingInterval, waitTimeout)
	})
})
