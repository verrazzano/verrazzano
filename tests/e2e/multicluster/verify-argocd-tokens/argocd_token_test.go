// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd_token_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

const (
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
	argoCDNamespace = "argocd"
)

var t = framework.NewTestFramework("cluster_sync_test")
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var sugarredLogger := 

var _ = t.Describe("ArgoCD Token Sync Testing", Label("f:platform-lcm.install"), func() {
	t.It("has the expected secrets", func() {
		secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
		Eventually(func() error {
			result, err := findSecret(argoCDNamespace, secretName)
			if result != false {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", secretName, argoCDNamespace, err))
				return err
			}
			return err
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to find secret "+secretName)
	})
	t.It("The expected secret currently contains both the createdAt and ExpiredAt annotations", func() {
		secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
		Eventually(func() error {
			result, err := verifyCreatedAtAndExpiresAtTimestampsExist(argoCDNamespace, secretName)
			if result != true {
				pkg.Log(pkg.Error, fmt.Sprintf("Failed to get an ExpiredAt or Created Annotation in secret %s in namespace %s with error: %v", secretName, argoCDNamespace, err))
				return err
			}
			return err
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Expected to find both Created and Expired At Annotations "+secretName)
	})
	t.It("A new ArgoCD token is able to be created through the Rancher API", func() {
		pkg.CreateNewRancherConfigForUser(log)
	})

})

func findSecret(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		return false, err
	}
	return s != nil, nil
}

func verifyCreatedAtAndExpiresAtTimestampsExist(namespace, name string) (bool, error) {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return false, err
	}
	annotationMap := s.GetAnnotations()
	createdValue, ok := annotationMap["verrazzano.io/create-timestamp"]
	if !ok || createdValue == "" {
		return false, fmt.Errorf("Created Annotation Value Not Found")
	}
	expiresAtValue, ok := annotationMap["verrazzano.io/expires-at-timestamp"]
	if !ok || expiresAtValue == "" {
		return false, fmt.Errorf("Expiration Value is Not Found")
	}
	return true, nil
}
