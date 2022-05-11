// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package env_dns_cert

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
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
	t                          = framework.NewTestFramework("update dns")
	currentEnvironmentName     string
	currentDNSDomain           string
	currentCertName            string = "verrazzano-ca-certificate"
	currentCertNamespace       string = "cert-manager"
	currentCertSecretName      string = "verrazzano-ca-certificate-secret"
	currentCertSecretNamespace string = "cert-manager"
	currentCertIssuerName      string = "verrazzano-selfsigned-issuer"
	currentCertIssuerNamespace string = "cert-manager"

	testEnvironmentName     string = "test-env"
	testDNSDomain           string = "sslip.io"
	testCertName            string = "test-ca"
	testCertSecretName      string = "test-secret-ca"
	testCertSecretNamespace string = "test-namespace"
	testCertIssuerName      string = "verrazzano-cluster-issuer"
)

var _ = t.AfterSuite(func() {
	files := []string{testCertName + ".crt", testCertName + ".key"}
	cleanupTemporaryFiles(files)
})

var _ = t.Describe("Test updates to environment name, dns domain and cert-manager CA certificates", func() {
	t.It("Verify the current environment name", func() {
		cr := update.GetCR()
		currentEnvironmentName = cr.Spec.EnvironmentName
		currentDNSDomain = cr.Spec.Components.DNS.Wildcard.Domain
		validateIngressList(currentEnvironmentName, currentDNSDomain)
		validateVirtualServiceList(currentDNSDomain)
	})

	t.It("Update and verify environment name", func() {
		m := EnvironmentNameModifier{testEnvironmentName}
		update.UpdateCR(m)
		validateIngressList(testEnvironmentName, currentDNSDomain)
		validateVirtualServiceList(currentDNSDomain)
	})

	t.It("Update and verify dns domain", func() {
		m := WildcardDNSModifier{testDNSDomain}
		update.UpdateCR(m)
		validateIngressList(testEnvironmentName, testDNSDomain)
		validateVirtualServiceList(testDNSDomain)
	})

	t.It("Update and verify CA certificate", func() {
		createCustomCACertificate(testCertName, testCertSecretNamespace, testCertSecretName)
		m := CustomCACertificateModifier{testCertSecretNamespace, testCertSecretName}
		update.UpdateCR(m)
		validateCertManagerResourcesCleanup()
		validateCACertificateIssuer(testCertIssuerName)
	})
})

func validateIngressList(environmentName string, domain string) {
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
	Eventually(func() bool {
		// Fetch the virtual services for the deployed applications
		virtualServiceList, err := pkg.GetVirtualServiceList("")
		if err != nil {
			log.Fatalf("Error while fetching GatewayList\n%s", err)
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
	output, err := exec.Command("/bin/sh", "create-custom-ca.sh", "-k", "-c", certName, "-s", secretName, "-n", secretNamespace).Output()
	if err != nil {
		log.Println("Error in creating custom CA secret using the script create-custom-ca.sh")
		log.Fatalf("Arguments:\n\t Certificate name: %s\n\t Secret name: %s\n\t Secret namespace: %s\n", certName, secretName, secretNamespace)
	}
	log.Println(string(output))
}

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

func validateCertManagerResourcesCleanup() {
	Eventually(func() bool {
		// Verify that the existing certificate has been removed
		certificateList, err := pkg.GetCertificateList(currentCertNamespace)
		if err != nil {
			log.Fatalf("Error while fetching CertificateList\n%s", err)
		}
		for _, certificate := range certificateList.Items {
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
		if err == nil {
			log.Printf("Secret %s should NOT exist in the namespace %s\n", currentCertSecretName, currentCertSecretNamespace)
			return false
		}
		return true
	}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected that the default CA resources should be cleaned up")
}

func cleanupTemporaryFiles(files []string) error {
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
