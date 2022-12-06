// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tools/psr/tests/pkg/constants"
)

func CheckScenarioPods(t *framework.TestFramework, log vzlog.VerrazzanoLogger, namespace string, scenarioPods [][]string) {
	testfunc := func(name string, expected bool) {
		gomega.Eventually(func() (bool, error) {
			exists, err := pkg.DoesPodExist(namespace, name)
			if exists {
				t.Logs.Infof("Found pod %s/%s", namespace, name)
			}
			return exists, err
		}, constants.WaitTimeout, constants.PollingInterval).Should(gomega.Equal(expected))
	}

	podTableEntries := []interface{}{testfunc}
	for i := range scenarioPods {
		testDesc := scenarioPods[i][0]
		podName := scenarioPods[i][1]
		tableEntry := ginkgo.Entry(testDesc, podName, true)
		podTableEntries = append(podTableEntries, tableEntry)
	}

	// GIVEN a Verrazzano installation with a running PSR ops-s2 scenario
	// WHEN  we wish to validate the PSR workers
	// THEN  the scenario pods are running
	t.DescribeTable("Scenario pods are deployed,", podTableEntries...)
}
