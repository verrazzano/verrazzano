// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak_test

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"path"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	keycloakNamespace string = "keycloak"
)

var volumeClaims map[string]*corev1.PersistentVolumeClaim

var _ = ginkgo.Describe("Verify Keycloak configuration", func() {
	var _ = ginkgo.Context("Verify password policies", func() {
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

var _ = ginkgo.Describe("Verify MySQL Persistent Volumes based on install profile", func() {
	var _ = ginkgo.Context("Verify Persistent volumes allocated per install profile", func() {

		var err error
		size := "50Gi"

		volumeClaims, err = pkg.GetPersistentVolumes(keycloakNamespace)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Error retrieving persistent volumes for verrazzano-system: %v", err))
		}

		if pkg.IsDevProfile() {
			ginkgo.It("Verify persistent volumes in namespace keycloak based on Dev install profile", func() {
				// There is no Persistent Volume for MySQL in a dev install
				gomega.Expect(len(volumeClaims)).To(gomega.Equal(0))
			})
		} else if pkg.IsManagedClusterProfile() {
			ginkgo.It("Verify namespace keycloak doesn't exist based on Managed Cluster install profile", func() {
				// There is no keycloak namespace in a managed cluster install
				ns, _ := pkg.GetNamespace(keycloakNamespace)
				gomega.Expect(ns.Name).To(gomega.BeEmpty())
			})
		} else if pkg.IsProdProfile() {
			ginkgo.It("Verify persistent volumes in namespace keycloak based on Prod install profile", func() {
				// 50 GB Persistent Volume create for MySQL in a prod install
				gomega.Expect(len(volumeClaims)).To(gomega.Equal(1))
				assertPersistentVolume("mysql", size)
			})
		}
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

func assertPersistentVolume(key string, size string) {
	gomega.Expect(volumeClaims).To(gomega.HaveKey(key))
	pvc := volumeClaims[key]
	gomega.Expect(pvc.Spec.Resources.Requests.Storage().String()).To(gomega.Equal(size))
}
