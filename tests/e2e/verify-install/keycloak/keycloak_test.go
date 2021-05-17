// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak_test

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout     = 10 * time.Minute
	pollingInterval = 30 * time.Second
)

var _ = ginkgo.Describe("Verify Keycloak configuration", func() {
	var _ = ginkgo.Describe("Verify password policies", func() {
		isManagedClusterProfile := pkg.IsManagedClusterProfile()
		ginkgo.It("Verify master realm password policy", func() {
			if !isManagedClusterProfile {
				// GIVEN the password policy setup for the master realm during installation
				// WHEN valid and invalid password changes are attempted
				// THEN verify valid passwords are accepted and invalid passwords are rejected.
				gomega.Eventually(verifyKeycloakMasterRealmPasswordPolicyIsCorrect, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			}
		})
		ginkgo.It("Verify verrazzano-system realm password policy", func() {
			if !isManagedClusterProfile {
				// GIVEN the password policy setup for the verrazzano-system realm during installation
				// WHEN valid and invalid password changes are attempted
				// THEN verify valid passwords are accepted and invalid passwords are rejected.
				gomega.Eventually(verifyKeycloakVerrazzanoRealmPasswordPolicyIsCorrect, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			}
		})
	})
})

func verifyKeycloakVerrazzanoRealmPasswordPolicyIsCorrect() bool {
	return verifyKeycloakRealmPasswordPolicyIsCorrect("verrazzano-system")
}

func verifyKeycloakMasterRealmPasswordPolicyIsCorrect() bool {
	return verifyKeycloakRealmPasswordPolicyIsCorrect("master")
}

func verifyKeycloakRealmPasswordPolicyIsCorrect(realm string) bool {
	kc, err := pkg.NewKeycloakAdminRESTClient()
	if err != nil {
		fmt.Printf("Failed to create Keycloak REST client: %v\n", err)
		return false
	}

	var realmData map[string]interface{}
	realmData, err = kc.GetRealm(realm)
	if err != nil {
		fmt.Printf("Failed to get realm %s\n", realm)
		return false
	}
	if realmData["passwordPolicy"] == nil {
		fmt.Printf("Failed to find password policy for realm: %s\n", realm)
		return false
	}
	policy := realmData["passwordPolicy"].(string)
	if len(policy) == 0 || !strings.Contains(policy, "length") {
		fmt.Printf("Failed to find password policy for realm: %s\n", realm)
		return false
	}

	salt := time.Now().Format("20060102150405.000000000")
	userName := fmt.Sprintf("test-user-%s", salt)
	firstName := fmt.Sprintf("test-first-%s", salt)
	lastName := fmt.Sprintf("test-last-%s", salt)
	validPassword := fmt.Sprintf("test-password-12-!@-AB-%s", salt)
	userURL, err := kc.CreateUser(realm, userName, firstName, lastName, validPassword)
	if err != nil {
		fmt.Printf("Failed to create user %s/%s: %v\n", realm, userName, err)
		return false
	}
	userID := path.Base(userURL)
	defer func() {
		err = kc.DeleteUser(realm, userID)
		if err == nil {
			fmt.Printf("Failed to delete user %s/%s: %v\n", realm, userID, err)
		}
	}()
	err = kc.SetPassword(realm, userID, "invalid")
	if err == nil {
		fmt.Printf("Should not have been able to set password for %s/%s\n", realm, userID)
		return false
	}
	newValidPassword := fmt.Sprintf("test-new-password-12-!@-AB-%s", salt)
	err = kc.SetPassword(realm, userID, newValidPassword)
	if err != nil {
		fmt.Printf("Failed to set password for %s/%s: %v\n", realm, userID, err)
		return false
	}
	return true
}
