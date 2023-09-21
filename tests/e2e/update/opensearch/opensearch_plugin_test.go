// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package opensearch

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

const osMasterNodegroup = "es-master"

var _ = t.Describe("Update Plugins", Label("f:platform-plugin.update"), func() {

	// GIVEN a VZ custom resource in dev profile,
	// WHEN opensearch plugin is updated with wrong value
	// THEN master pods gets in failure state.
	// In last opensearch plugin is disabled
	// Then all pods are back to normal state
	t.It("opensearch update plugin", func() {
		m := OpenSearchPlugins{Enabled: true, InstanceList: "abc"}
		update.UpdateCRWithPlugins(m, pollingInterval, waitTimeout)
		update.ValidatePods(osMasterNodegroup, NodeGroupLabel, constants.VerrazzanoSystemNamespace, 0, false)
		m = OpenSearchPlugins{Enabled: false, InstanceList: "analysis-stempel"}
		update.UpdateCRWithPlugins(m, pollingInterval, waitTimeout)
		var pods []corev1.Pod
		var err error
		Eventually(func() error {
			pods, err = pkg.GetPodsFromSelector(&v1.LabelSelector{MatchLabels: map[string]string{NodeGroupLabel: osMasterNodegroup}}, constants.VerrazzanoSystemNamespace)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return err
			}
			return nil
		}).WithPolling(20*time.Second).WithTimeout(2*time.Minute).Should(BeNil(), "failed to fetch the opensearch master pods")
		update.ValidatePods(osMasterNodegroup, NodeGroupLabel, constants.VerrazzanoSystemNamespace, uint32(len(pods)), false)
	})
})
