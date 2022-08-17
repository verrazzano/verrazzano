// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var waitTimeout = 4 * time.Minute
var pollingInterval = 30 * time.Second

var t = framework.NewTestFramework("verify")

var userIDConfig map[string]string

var _ = t.BeforeSuite(func() {
	start := time.Now()
	Fail("force failing the tests")
	beforeSuitePassed = true

	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		Skip("Skipping test suite since this is a managed cluster profile")
	}

	exists, err := pkg.DoesNamespaceExist(pkg.TestKeycloakNamespace)
	if err != nil {
		Fail(err.Error())
	}
	if !exists {
		AbortSuite(fmt.Sprintf("Skipping test suite since the namespace %s does not exist", pkg.TestKeycloakNamespace))
	}

	configMap, err := pkg.GetConfigMap(pkg.TestKeycloakConfigMap, pkg.TestKeycloakNamespace)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed getting configmap: %v", err))
	}

	if configMap == nil {
		AbortSuite(fmt.Sprintf("Skipping test suite since the configmap %s does not exist", pkg.TestKeycloakConfigMap))
	}

	userIDConfigData := configMap.Data["data"]
	err = yaml.Unmarshal([]byte(userIDConfigData), &userIDConfig)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed getting configmap data: %v", err))
	}

	metrics.Emit(t.Metrics.With("before_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	start := time.Now()
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Verify users exist in Keycloak", Label("f:platform-lcm.install"), func() {
	t.It("Verifying user in master realm", func() {
		Eventually(verifyUserExistsMaster, waitTimeout, pollingInterval).ShouldNot(BeTrue())
	})
	t.It("Verifying user in verrazzano-system realm", func() {
		Eventually(verifyUserExistsVerrazzano, waitTimeout, pollingInterval).ShouldNot(BeTrue())
	})
})

func verifyUserExistsMaster() bool {
	return false
	//return verifyUserExists("master", userIDConfig[pkg.TestKeycloakMasterUserIDKey])
}

func verifyUserExistsVerrazzano() bool {
	return false
	//return verifyUserExists("verrazzano-system", userIDConfig[pkg.TestKeycloakVerrazzanoUserIDKey])
}

func verifyUserExists(realm, userID string) bool {
	kc, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Error(fmt.Printf("Failed to create Keycloak REST client: %v\n", err))
		return false
	}

	exists, err := kc.VerifyUserExists(realm, userID)
	if err != nil {
		t.Logs.Info(fmt.Printf("Failed to verify user %s/%s: %v\n", realm, userID, err))
		return false
	}
	return exists
}
