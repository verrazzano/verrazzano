// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
	"os/exec"
	"path"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
	v1 "k8s.io/api/core/v1"
)

var waitTimeout = 10 * time.Minute
var pollingInterval = 20 * time.Second

var kubeConfig = os.Getenv("KUBECONFIG")
var testKeycloakMasterUserID = ""
var testKeycloakVerrazzanoUserID = ""

var t = framework.NewTestFramework("keycloak")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	beforeSuitePassed = true

	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		Skip("Skipping test suite since this is a managed cluster profile")
	}

	exists, err := pkg.DoesNamespaceExist(pkg.TestKeycloakNamespace)
	if err != nil {
		Fail(err.Error())
	}

	if exists {
		t.Logs.Info("Delete namespace")
		// Delete namespace, if already exists, so that test can be executed cleanly
		Eventually(func() error {
			return pkg.DeleteNamespace(pkg.TestKeycloakNamespace)
		}, waitTimeout, pollingInterval).Should(BeNil())

		t.Logs.Info("Wait for namespace finalizer to be removed")
		Eventually(func() bool {
			return pkg.CheckNamespaceFinalizerRemoved(pkg.TestKeycloakNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue())

		t.Logs.Info("Wait for namespace deletion")
		Eventually(func() bool {
			_, err := pkg.GetNamespace(pkg.TestKeycloakNamespace)
			return err != nil && errors.IsNotFound(err)
		}, waitTimeout, pollingInterval).Should(BeTrue())
	}

	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{}
		return pkg.CreateNamespace(pkg.TestKeycloakNamespace, nsLabels)
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())

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
	createConfigMap()
	metrics.Emit(t.Metrics.With("after_suite_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("Create users in Keycloak", Label("f:platform-lcm.install"), func() {
	t.It("Creating user in master realm", func() {
		Eventually(verifyCreateUserMaster, waitTimeout, pollingInterval).Should(Not(BeNil()))
	})
	t.It("Creating user in verrazzano-system realm", func() {
		Eventually(verifyCreateUserVz, waitTimeout, pollingInterval).Should(Not(BeNil()))
	})
})

// Creating a configmap to store the newly created keycloak user ids to be verified in the keycloak post-upgrade later
func createConfigMap() {
	if testKeycloakMasterUserID != "" && testKeycloakVerrazzanoUserID != "" {
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
	}
}

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
