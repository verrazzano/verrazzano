// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"fmt"
	"os"
	"os/exec"
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

var kubeConfig = os.Getenv("KUBECONFIG")
var testKeycloakMasterUserID = ""
var testKeycloakVerrazzanoUserID = ""

var t = framework.NewTestFramework("verify")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("before_suite_elapsed_time", time.Since(start).Milliseconds()))

	kubeConfigOption := fmt.Sprintf("--kubeconfig=%s", kubeConfig)
	cmd := exec.Command("kubectl", kubeConfigOption, "create", "ns", pkg.TestKeycloakNamespace)
	t.Logs.Info(fmt.Sprintf("kubectl command to create namespace %s: %s", pkg.TestKeycloakNamespace, cmd.String()))
	out, err := cmd.Output()
	if len(string(out)) != 0 {
		t.Logs.Info(fmt.Printf("Output while creating namespace %s: %s", pkg.TestKeycloakNamespace, string(out)))
	}
	if err != nil {
		AbortSuite(fmt.Sprintf("Error creating namespace %s: %s\n", pkg.TestKeycloakNamespace, err))
	}
})

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || framework.VzCurrentGinkgoTestDescription().Failed()
})

var _ = t.AfterSuite(func() {
	start := time.Now()

	// Creating a configmap to store the newly created keycloak user ids to be verified in the keycloak post-upgrade later
	kubeConfigOption := fmt.Sprintf("--kubeconfig=%s", kubeConfig)
	keyValue1 := fmt.Sprintf("--from-literal=%s=%s", pkg.TestKeycloakMasterUserIDKey, testKeycloakMasterUserID)
	keyValue2 := fmt.Sprintf("--from-literal=%s=%s", pkg.TestKeycloakVerrazzanoUserIDKey, testKeycloakVerrazzanoUserID)
	cmd := exec.Command("kubectl", kubeConfigOption, "-n", pkg.TestKeycloakNamespace,
		"create", "configmap", pkg.TestKeycloakConfigMap, keyValue1, keyValue2)
	t.Logs.Info(fmt.Sprintf("kubectl command to create configmap %s: %s", pkg.TestKeycloakConfigMap, cmd.String()))
	_, err := cmd.Output()
	if err != nil {
		t.Fail(fmt.Sprintf("Error creating configmap %s: %s\n", pkg.TestKeycloakConfigMap, err))
	}
	if failed || !beforeSuitePassed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Create users in Keycloak", Label("f:platform-lcm.install"), func() {
	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	t.It("Creating user in master realm", func() {
		if !isManagedClusterProfile {
			Eventually(verifyCreateUserMaster, waitTimeout, pollingInterval).Should(Not(BeNil()))
		}
	})
	t.It("Creating user in verrazzano-system realm", func() {
		if !isManagedClusterProfile {
			Eventually(verifyCreateUserVz, waitTimeout, pollingInterval).Should(Not(BeNil()))
		}
	})
})

func verifyCreateUserMaster() (string, error) {
	userID, err := verifyCreateUser("master")
	testKeycloakMasterUserID = userID
	return userID, err
}

func verifyCreateUserVz() (string, error) {
	userID, err := verifyCreateUser("verrazzano-system")
	testKeycloakVerrazzanoUserID = userID
	return userID, err
}

func verifyCreateUser(realm string) (string, error) {
	kc, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		t.Logs.Error(fmt.Printf("Failed to create Keycloak REST client: %v\n", err))
		return "", err
	}

	salt := time.Now().Format("20060102150405.000000000")
	userName := fmt.Sprintf("test-user-%s", salt)
	firstName := fmt.Sprintf("test-first-%s", salt)
	lastName := fmt.Sprintf("test-last-%s", salt)
	validPassword := fmt.Sprintf("test-password-12-!@-AB-%s", salt)
	userURL, err := kc.CreateUser(realm, userName, firstName, lastName, validPassword)
	if err != nil {
		t.Logs.Error(fmt.Printf("Failed to create user %s/%s: %v\n", realm, userName, err))
		return "", err
	}
	userID := path.Base(userURL)
	return userID, err
}
