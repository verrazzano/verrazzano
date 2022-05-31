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

type ACMECertificateModifier struct {
	EmailAddress string
	Environment  string
}

func (u ACMECertificateModifier) ModifyCR(cr *vzapi.Verrazzano) {
	var b bool = true
	cr.Spec.Components.CertManager = &vzapi.CertManagerComponent{}
	cr.Spec.Components.CertManager.Enabled = &b
	cr.Spec.Components.CertManager.Certificate.Acme.Provider = vzapi.LetsEncrypt
	cr.Spec.Components.CertManager.Certificate.Acme.EmailAddress = u.EmailAddress
	cr.Spec.Components.CertManager.Certificate.Acme.Environment = u.Environment
}

var (
	t                           = framework.NewTestFramework("update cm (let's encrypt)")
	testAcmeEmailAddress string = "emailAddress@domain.com"
	testAcmeEnvironment  string = "staging"
)

var _ = t.Describe("Test updates cert-manager CA certificates", func() {
	t.It("Update and verify CA certificate", func() {
		m := ACMECertificateModifier{testAcmeEmailAddress, testAcmeEnvironment}
		err := update.UpdateCR(m)
		if err != nil {
			log.Fatalf("Error in updating CA certificate - %s", err)
		}
		validateCACertificateIssuer("lets-encrypt")
	})
})

func validateCACertificateIssuer(certIssuer string) {
	Eventually(func() bool {
		// Fetch the certificates for the deployed applications
		certificateList, err := pkg.GetCertificateList("")
		if err != nil {
			log.Fatalf("Error while fetching CertificateList\n%s", err)
		}
		// Verify that the certificate is issued by the right cluster issuer
		for _, certificate := range certificateList.Items {
			currIssuer := certificate.Spec.IssuerRef.Name
			if currIssuer != certIssuer {
				log.Printf("Issuer for the certificate %s in namespace %s is %s; expected is %s\n", certificate.Name, certificate.Namespace, currIssuer, certIssuer)
				return false
			}
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the certificates have a valid issuer")
}
