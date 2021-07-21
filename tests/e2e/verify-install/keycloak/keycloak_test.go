// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak_test

import (
	"fmt"
	"path"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	keycloakNamespace string = "keycloak"
)

var volumeClaims map[string]*corev1.PersistentVolumeClaim

var _ = BeforeSuite(func() {
	Eventually(func() (map[string]*corev1.PersistentVolumeClaim, error) {
		var err error
		volumeClaims, err = pkg.GetPersistentVolumes(keycloakNamespace)
		return volumeClaims, err
	}, waitTimeout, pollingInterval).ShouldNot(BeNil())
})

var _ = Describe("Verify Keycloak configuration", func() {
	var _ = Context("Verify password policies", func() {
		profile, err := pkg.GetVerrazzanoProfile()
		Expect(err).To(BeNil())
		It("Verify master realm password policy", func() {
			if *profile != v1alpha1.ManagedCluster {
				// GIVEN the password policy setup for the master realm during installation
				// WHEN valid and invalid password changes are attempted
				// THEN verify valid passwords are accepted and invalid passwords are rejected.
				Eventually(verifyKeycloakMasterRealmPasswordPolicyIsCorrect, waitTimeout, pollingInterval).Should(BeTrue())
			}
		})
		It("Verify verrazzano-system realm password policy", func() {
			if *profile != v1alpha1.ManagedCluster {
				// GIVEN the password policy setup for the verrazzano-system realm during installation
				// WHEN valid and invalid password changes are attempted
				// THEN verify valid passwords are accepted and invalid passwords are rejected.
				Eventually(verifyKeycloakVerrazzanoRealmPasswordPolicyIsCorrect, waitTimeout, pollingInterval).Should(BeTrue())
			}
		})
	})
})

var _ = Describe("Verify MySQL Persistent Volumes based on install profile", func() {
	var _ = Context("Verify Persistent volumes allocated per install profile", func() {

		const size = "8Gi" // based on values set in platform-operator/thirdparty/charts/mysql

		profile, err := pkg.GetVerrazzanoProfile()
		Expect(err).To(BeNil())
		if *profile == v1alpha1.Dev {
			It("Verify persistent volumes in namespace keycloak based on Dev install profile", func() {
				// There is no Persistent Volume for MySQL in a dev install
				Expect(len(volumeClaims)).To(Equal(0))
			})
		} else if *profile == v1alpha1.ManagedCluster {
			It("Verify namespace keycloak doesn't exist based on Managed Cluster install profile", func() {
				// There is no keycloak namespace in a managed cluster install
				Eventually(func() bool {
					_, err := pkg.GetNamespace(keycloakNamespace)
					return err != nil && errors.IsNotFound(err)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		} else if *profile == v1alpha1.Prod {
			It("Verify persistent volumes in namespace keycloak based on Prod install profile", func() {
				// 50 GB Persistent Volume create for MySQL in a prod install
				Expect(len(volumeClaims)).To(Equal(1))
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
	Expect(volumeClaims).To(HaveKey(key))
	pvc := volumeClaims[key]
	Expect(pvc.Spec.Resources.Requests.Storage().String()).To(Equal(size))
}
