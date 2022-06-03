// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cmletsencrypt

import (
	"log"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
)

const (
	waitTimeout     = 15 * time.Minute
	pollingInterval = 10 * time.Second
)

type ACMECertManagerModifier struct {
	EmailAddress string
	Environment  string
}

func (u ACMECertManagerModifier) ModifyCR(cr *vzapi.Verrazzano) {
	var b bool = true
	cr.Spec.Components.CertManager = &vzapi.CertManagerComponent{}
	cr.Spec.Components.CertManager.Enabled = &b
	cr.Spec.Components.CertManager.Certificate.Acme.Provider = vzapi.LetsEncrypt
	cr.Spec.Components.CertManager.Certificate.Acme.EmailAddress = u.EmailAddress
	cr.Spec.Components.CertManager.Certificate.Acme.Environment = u.Environment
}

var (
	t                           = framework.NewTestFramework("Update Let's Encrypt CM")
	testAcmeEmailAddress string = "emailAddress@domain.com"
	testAcmeEnvironment  string = "staging"

	clusterIssuerName           string = "verrazzano-cluster-issuer"
	clusterIssuerPrivateKeyName string = "verrazzano-cert-acme-secret"
)

var _ = t.Describe("Test updates cert-manager CA certificates", func() {
	t.It("Update and verify CA certificate", func() {
		m := ACMECertManagerModifier{testAcmeEmailAddress, testAcmeEnvironment}
		err := update.UpdateCR(m)
		if err != nil {
			log.Fatalf("Error in updating CA - %s", err)
		}
		validateClusterIssuerUpdate()
	})
})

func validateClusterIssuerUpdate() {
	log.Printf("Validating updates to the ClusterIssuer")
	Eventually(func() bool {
		// Fetch the cluster issuers
		clusterIssuer, err := pkg.GetClusterIssuer(clusterIssuerName)
		if err != nil {
			log.Fatalf("Error while fetching ClusterIssuers\n%s", err)
		}
		// Verify that the cluster issuer has been updated
		if clusterIssuer.Spec.ACME == nil {
			log.Printf("ClusterIssuer %s does not contain ACME section\n", clusterIssuerName)
			return false
		}
		if clusterIssuer.Spec.ACME.Email != testAcmeEmailAddress {
			log.Printf("ClusterIssuer ACME section contains the email %s, instead of the email %s\n", clusterIssuer.Spec.ACME.Email, testAcmeEmailAddress)
			return false
		}
		if clusterIssuer.Spec.ACME.PrivateKey.Name != clusterIssuerPrivateKeyName {
			log.Printf("ClusterIssuer ACME section contains the private key %s, instead of the private key %s\n", clusterIssuer.Spec.ACME.PrivateKey.Name, clusterIssuerPrivateKeyName)
			return false
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the cluster issuer should be updated")
}
