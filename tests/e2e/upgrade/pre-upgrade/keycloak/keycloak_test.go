// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"fmt"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

var waitTimeout = 1 * time.Minute
var pollingInterval = 30 * time.Second

var t = framework.NewTestFramework("verify")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	beforeSuitePassed = true
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
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Create users in Keycloak", Label("f:platform-lcm.install"), func() {
	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	t.It("Creating user in master realm", func() {
		if !isManagedClusterProfile {
			Eventually(verifyCreateUserMaster, waitTimeout, pollingInterval).Should(BeTrue())
		}
	})
	t.It("Creating user in verrazzano-system realm", func() {
		if !isManagedClusterProfile {
			Eventually(verifyCreateUserVz, waitTimeout, pollingInterval).Should(BeTrue())
		}
	})
})

func verifyCreateUserMaster() bool {
	return verifyCreateUser("master", pkg.TestKeycloakMasterUserIdEnvVar)
}

func verifyCreateUserVz() bool {
	return verifyCreateUser("verrazzano-system", pkg.TestKeycloakVerrazzanoUserIdEnvVar)
}

func verifyCreateUser(realm, userIDEnvVar string) bool {
	kc, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Error(fmt.Printf("Failed to create Keycloak REST client: %v\n", err))
		return false
	}

	salt := time.Now().Format("20060102150405.000000000")
	userName := fmt.Sprintf("test-user-%s", salt)
	firstName := fmt.Sprintf("test-first-%s", salt)
	lastName := fmt.Sprintf("test-last-%s", salt)
	validPassword := fmt.Sprintf("test-password-12-!@-AB-%s", salt)
	userURL, err := kc.CreateUser(realm, userName, firstName, lastName, validPassword)
	if err != nil {
		t.Logs.Error(fmt.Printf("Failed to create user %s/%s: %v\n", realm, userName, err))
		return false
	}
	userID := path.Base(userURL)
	os.Setenv(userIDEnvVar, userID)
	return true
}
