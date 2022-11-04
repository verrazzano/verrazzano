// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package envdnscm

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	. "github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
)

type EnvironmentNameModifier struct {
	EnvironmentName string
}

type WildcardDNSModifier struct {
	Domain string
}

type CustomCACertificateModifier struct {
	ClusterResourceNamespace string
	SecretName               string
}

func (u EnvironmentNameModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.EnvironmentName = u.EnvironmentName
}

func (u WildcardDNSModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.DNS == nil {
		cr.Spec.Components.DNS = &vzapi.DNSComponent{}
	}
	if cr.Spec.Components.DNS.Wildcard == nil {
		cr.Spec.Components.DNS.Wildcard = &vzapi.Wildcard{}
	}
	cr.Spec.Components.DNS.Wildcard.Domain = u.Domain
}

func (u CustomCACertificateModifier) ModifyCR(cr *vzapi.Verrazzano) {
	if cr.Spec.Components.CertManager == nil {
		cr.Spec.Components.CertManager = &vzapi.CertManagerComponent{}
	}
	cr.Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace = u.ClusterResourceNamespace
	cr.Spec.Components.CertManager.Certificate.CA.SecretName = u.SecretName
}

var (
	t                       = framework.NewTestFramework("update env-dns-cm")
	testEnvironmentName     = "test-env"
	testDNSDomain           = "sslip.io"
	testCertName            = "test-ca"
	testCertSecretName      = "test-secret-ca"
	testCertSecretNamespace = "test-namespace"
	testCertIssuerName      = "verrazzano-cluster-issuer"

	currentEnvironmentName     string
	currentDNSDomain           string
	currentCertNamespace       = "cert-manager"
	currentCertName            = "verrazzano-ca-certificate"
	currentCertIssuerNamespace = "cert-manager"
	currentCertIssuerName      = "verrazzano-selfsigned-issuer"
	currentCertSecretNamespace = "cert-manager"
	/* #nosec G101 -- This is a false positive */
	currentCertSecretName = "verrazzano-ca-certificate-secret"
)

var _ = t.AfterSuite(func() {
	files := []string{testCertName + ".crt", testCertName + ".key"}
	cleanupTemporaryFiles(files)
})

var _ = t.Describe("Test updates to environment name, dns domain and cert-manager CA certificates", func() {
	t.It("Verify the current environment name", func() {
		cr := update.GetCR()
		currentEnvironmentName = pkg.GetEnvironmentName(cr)
		currentDNSDomain = pkg.GetDNS(cr)
		validateIngressList(currentEnvironmentName, currentDNSDomain)
		validateVirtualServiceList(currentDNSDomain)
	})

	t.It("Update and verify environment name", func() {
		m := EnvironmentNameModifier{testEnvironmentName}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validateIngressList(testEnvironmentName, currentDNSDomain)
		validateVirtualServiceList(currentDNSDomain)
		pkg.VerifyKeycloakAccess(t.Logs)
		pkg.VerifyRancherAccess(t.Logs)
		pkg.VerifyRancherKeycloakAuthConfig(t.Logs)
	})

	t.It("Update and verify dns domain", func() {
		m := WildcardDNSModifier{testDNSDomain}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validateIngressList(testEnvironmentName, testDNSDomain)
		validateVirtualServiceList(testDNSDomain)
		pkg.VerifyKeycloakAccess(t.Logs)
		pkg.VerifyRancherAccess(t.Logs)
		pkg.VerifyRancherKeycloakAuthConfig(t.Logs)
	})

	t.It("Update and verify CA certificate", func() {
		createCustomCACertificate(testCertName, testCertSecretNamespace, testCertSecretName)
		m := CustomCACertificateModifier{testCertSecretNamespace, testCertSecretName}
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
		validateCertManagerResourcesCleanup()
		validateCACertificateIssuer()
	})
})

func validateIngressList(environmentName string, domain string) {
	log.Printf("Validating the ingresses")
	Eventually(func() bool {
		// Fetch the ingresses for the Verrazzano components
		ingressList, err := pkg.GetIngressList("")
		if err != nil {
			log.Fatalf("Error while fetching IngressList\n%s", err)
		}
		// Verify that the ingresses contain the expected environment name and domain name
		for _, ingress := range ingressList.Items {
			hostname := ingress.Spec.Rules[0].Host
			if !strings.Contains(hostname, environmentName) {
				log.Printf("Ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, environmentName)
				return false
			}
			if !strings.Contains(hostname, domain) {
				log.Printf("Ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, domain)
				return false
			}
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the ingress hosts contain the expected environment and domain names")
}

func validateVirtualServiceList(domain string) {
	log.Printf("Validating the virtual services")
	Eventually(func() bool {
		// Fetch the virtual services for the deployed applications
		virtualServiceList, err := pkg.GetVirtualServiceList("")
		if err != nil {
			log.Fatalf("Error while fetching VirtualServiceList\n%s", err)
		}
		// Verify that the virtual services contain the expected environment name and domain nameƒ
		for _, virtualService := range virtualServiceList.Items {
			hostname := virtualService.Spec.Hosts[0]
			if !strings.Contains(hostname, domain) {
				log.Printf("Virtual Service %s in namespace %s with hostname %s must contain %s\n", virtualService.Name, virtualService.Namespace, hostname, domain)
				return false
			}
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the application virtual service hosts contain the expected domain name")
}

func createCustomCACertificate(certName string, secretNamespace string, secretName string) {
	log.Printf("Creating custom CA certificate")
	output, err := exec.Command("/bin/sh", "create-custom-ca.sh", "-k", "-c", certName, "-s", secretName, "-n", secretNamespace).Output()
	if err != nil {
		log.Println("Error in creating custom CA secret using the script create-custom-ca.sh")
		log.Fatalf("Arguments:\n\t Certificate name: %s\n\t Secret name: %s\n\t Secret namespace: %s\n", certName, secretName, secretNamespace)
	}
	log.Println(string(output))
}

func fetchCACertificatesFromIssuer(certIssuer string) []certmanagerv1.Certificate {
	// Reintialize the certificate list
	var certificates []certmanagerv1.Certificate
	// Fetch the certificates for the deployed applications
	certificateList, err := pkg.GetCertificateList("")
	if err != nil {
		log.Fatalf("Error while fetching CertificateList\n%s", err)
	}
	// Filter out the certificates that are issued by the given issuer
	for _, certificate := range certificateList.Items {
		if certificate.Spec.IssuerRef.Name == certIssuer {
			certificates = append(certificates, certificate)
		}
	}
	return certificates
}

func validateCACertificateIssuer() {
	log.Printf("Validating the CA certificates")
	Eventually(func() bool {
		// Fetch the certificates
		var certificates []certmanagerv1.Certificate = fetchCACertificatesFromIssuer(testCertIssuerName)
		// Verify that the certificate is issued by the right cluster issuer
		for _, certificate := range certificates {
			if certificate.Spec.IssuerRef.Name != testCertIssuerName {
				log.Printf("Issuer for the certificate %s in namespace %s is %s; expected is %s\n", certificate.Name, certificate.Namespace, certificate.Spec.IssuerRef.Name, testCertIssuerName)
				return false
			}
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the certificates have a valid issuer")
}

func validateCertManagerResourcesCleanup() {
	log.Printf("Validating CA certificate resource cleanup")
	Eventually(func() bool {
		// Fetch the certificates
		var certificates []certmanagerv1.Certificate = fetchCACertificatesFromIssuer(currentCertIssuerName)
		for _, certificate := range certificates {
			if certificate.Name == currentCertName {
				log.Printf("Certificate %s should NOT exist in the namespace %s\n", currentCertName, currentCertNamespace)
				return false
			}
		}
		// Verify that the certificate issuer has been removed
		issuerList, err := pkg.GetIssuerList(currentCertIssuerNamespace)
		if err != nil {
			log.Fatalf("Error while fetching IssuerList\n%s", err)
		}
		for _, issuer := range issuerList.Items {
			if issuer.Name == currentCertIssuerName {
				log.Printf("Issuer %s should NOT exist in the namespace %s\n", currentCertIssuerName, currentCertIssuerNamespace)
				return false
			}
		}
		// Verify that the secret used for the default certificate has been removed
		_, err = pkg.GetSecret(currentCertSecretNamespace, currentCertSecretName)
		if err != nil {
			log.Printf("Expected that the secret %s should NOT exist in the namespace %s", currentCertSecretName, currentCertSecretNamespace)
		} else {
			log.Printf("Secret %s should NOT exist in the namespace %s\n", currentCertSecretName, currentCertSecretNamespace)
			return false
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the default CA resources should be cleaned up")
}

func cleanupTemporaryFiles(files []string) error {
	log.Printf("Cleaning up temporary files")
	var err error
	for _, file := range files {
		_, err = os.Stat(file)
		if os.IsNotExist(err) {
			log.Printf("File %s does not exist", file)
			continue
		}
		err = os.Remove(file)
		if err != nil {
			log.Fatalf("Error while cleaning up temporary file %s\n%s", file, err)
		}
	}
	return err
}
