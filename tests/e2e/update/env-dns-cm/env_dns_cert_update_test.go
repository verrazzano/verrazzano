// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package envdnscm

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	constants2 "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"go.uber.org/zap"
	netv1 "k8s.io/api/networking/v1"
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
	if cr.Spec.Components.ClusterIssuer == nil {
		cr.Spec.Components.ClusterIssuer = &vzapi.ClusterIssuerComponent{}
	}
	if cr.Spec.Components.ClusterIssuer.CA == nil {
		cr.Spec.Components.ClusterIssuer.CA = &vzapi.CAIssuer{}
	}
	cr.Spec.Components.ClusterIssuer.ClusterResourceNamespace = u.ClusterResourceNamespace
	cr.Spec.Components.ClusterIssuer.CA.SecretName = u.SecretName
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

var afterSuite = t.AfterSuiteFunc(func() {
	files := []string{testCertName + ".crt", testCertName + ".key"}
	cleanupTemporaryFiles(files)
})

var _ = AfterSuite(afterSuite)

var _ = t.Describe("Test updates to environment name, dns domain and cert-manager CA certificates", func() {

	t.Context("Verify the current environment name", func() {
		cr := update.GetCR()
		currentEnvironmentName = pkg.GetEnvironmentName(cr)
		currentDNSDomain = pkg.GetDNS(cr)
		ItvalidateIngressList(currentEnvironmentName, currentDNSDomain)
		ItvalidateVirtualServiceList(currentDNSDomain)
	})

	t.Context("Update and verify environment name", func() {
		m := EnvironmentNameModifier{testEnvironmentName}
		ItupdateCR(m)
		ItvalidateIngressList(testEnvironmentName, currentDNSDomain)
		ItvalidateVirtualServiceList(currentDNSDomain)
		ItverifyIngressAccess(t.Logs)
	})

	t.Context("Update and verify dns domain", func() {
		m := WildcardDNSModifier{testDNSDomain}
		ItupdateCR(m)
		ItvalidateIngressList(testEnvironmentName, testDNSDomain)
		ItvalidateVirtualServiceList(testDNSDomain)
		ItverifyIngressAccess(t.Logs)
	})

	t.Context("Update and verify CA certificate", func() {
		createCustomCACertificate(testCertName, testCertSecretNamespace, testCertSecretName)
		m := CustomCACertificateModifier{testCertSecretNamespace, testCertSecretName}
		ItupdateCR(m)
		ItvalidateCertManagerResourcesCleanup()
		ItvalidateCACertificateIssuer()
	})
})

func ItupdateCR(m update.CRModifier) {
	t.It("Update the Verrazzano CR", func() {
		update.UpdateCRWithRetries(m, pollingInterval, waitTimeout)
	})
}

func ItvalidateIngressList(environmentName string, domain string) {
	t.It("Expect Ingresses to contain the correct hostname and domain", func() {
		Eventually(func() error {
			ingressList, err := pkg.GetIngressList("")
			if err != nil {
				return err
			}

			// Verify that the ingresses contain the expected environment name and domain name
			for _, ingress := range ingressList.Items {
				err := validateIngress(environmentName, domain, ingress)
				if err != nil {
					return err
				}
			}
			return nil
		}).WithTimeout(waitTimeout).WithPolling(pollingInterval).ShouldNot(HaveOccurred())
	})
}

func validateIngress(environmentName string, domain string, ingress netv1.Ingress) error {
	// Verify that the ingress contains the expected environment name and domain name
	if ingress.Namespace == constants.RancherSystemNamespace && ingress.Name == "vz-"+constants2.RancherIngress {
		// If this is the copy of the Rancher ingress that VZ makes in order to retain access for the managed clusters
		// until DNS updates have been pushed out to them, this ingress should have the old DNS. Skip this ingress when
		// verifying that DNS was updated.
		return nil
	}
	hostname := ingress.Spec.Rules[0].Host
	if !strings.Contains(hostname, environmentName) {
		return fmt.Errorf("ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, environmentName)
	}
	if !strings.Contains(hostname, domain) {
		return fmt.Errorf("ingress %s in namespace %s with hostname %s must contain %s", ingress.Name, ingress.Namespace, hostname, domain)
	}
	return nil
}

func ItvalidateVirtualServiceList(domain string) {
	// Fetch the virtual services for the deployed applications
	t.It("Expect VirtualServices to contain the expected environment and domain name", func() {
		Eventually(func() error {
			virtualServiceList, err := pkg.GetVirtualServiceList("")
			if err != nil {
				return err
			}

			// Verify that the virtual services contain the expected environment name and domain name
			for _, virtualService := range virtualServiceList.Items {
				hostname := virtualService.Spec.Hosts[0]
				if !strings.Contains(hostname, domain) {
					return fmt.Errorf("virtual service %s in namespace %s with hostname %s must contain %s", virtualService.Name, virtualService.Namespace, hostname, domain)
				}
			}
			return nil
		}).WithTimeout(waitTimeout).WithPolling(pollingInterval).ShouldNot(HaveOccurred())
	})
}

func ItverifyIngressAccess(log *zap.SugaredLogger) {
	if log == nil {
		log = zap.S()
	}

	t.DescribeTable("Access Ingresses",
		func(access func() error) {
			Eventually(access).WithTimeout(waitTimeout).WithPolling(pollingInterval).ShouldNot(HaveOccurred())
		},
		Entry("Access Keycloak", func() error { return pkg.VerifyKeycloakAccess(log) }),
		Entry("Access Rancher", func() error { return pkg.VerifyRancherAccess(log) }),
		Entry("Access Rancher with Keycloak", func() error { return pkg.VerifyRancherKeycloakAuthConfig(log) }),
	)
}

func createCustomCACertificate(certName string, secretNamespace string, secretName string) {
	log.Printf("Creating custom CA certificate")
	output, err := exec.Command("/bin/sh", "create-custom-ca.sh", "-k", "-c", certName, "-s", secretName, "-n", secretNamespace).Output()
	Expect(err).ToNot(HaveOccurred())
	log.Println(string(output))
}

func fetchCACertificatesFromIssuer(certIssuer string) ([]certmanagerv1.Certificate, error) {
	// Reintialize the certificate list
	var certificates []certmanagerv1.Certificate
	// Fetch the certificates for the deployed applications
	certificateList, err := pkg.GetCertificateList("")
	if err != nil {
		return certificates, err
	}

	// Filter out the certificates that are issued by the given issuer
	for _, certificate := range certificateList.Items {
		if certificate.Spec.IssuerRef.Name == certIssuer {
			certificates = append(certificates, certificate)
		}
	}
	return certificates, nil
}

func ItvalidateCACertificateIssuer() {
	t.It("Validate CA Certificate Issuer is listed Certificates", func() {
		Eventually(func() error {
			// Fetch the certificates
			certificates, err := fetchCACertificatesFromIssuer(testCertIssuerName)
			if err != nil {
				return err
			}
			// Verify that the certificate is issued by the right cluster issuer
			for _, certificate := range certificates {
				if certificate.Spec.IssuerRef.Name != testCertIssuerName {
					return fmt.Errorf("issuer for the certificate %s in namespace %s is %s; expected is %s", certificate.Name, certificate.Namespace, certificate.Spec.IssuerRef.Name, testCertIssuerName)
				}
			}
			return nil
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	})
}

func ItvalidateCertManagerResourcesCleanup() {
	// Verify that the certificates have been removed
	t.It("Validate Certificate cleanup", func() {
		Eventually(func() error {
			// Fetch the certificates
			certificates, err := fetchCACertificatesFromIssuer(testCertIssuerName)
			if err != nil {
				return err
			}
			// Verify that the certificate is issued by the right cluster issuer
			for _, certificate := range certificates {
				if certificate.Name == currentCertName {
					return fmt.Errorf("certificate %s should NOT exist in the namespace %s", currentCertName, currentCertNamespace)
				}
			}
			return nil
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	})

	// Verify that the certificate issuer has been removed
	t.It("Validate Issuer cleanup", func() {
		Eventually(func() error {
			// Fetch the certificates
			issuerList, err := pkg.GetIssuerList(currentCertIssuerNamespace)
			if err != nil {
				return err
			}
			// Verify that the certificate is issued by the right cluster issuer
			for _, issuer := range issuerList.Items {
				if issuer.Name != currentCertIssuerName {
					return fmt.Errorf("issuer %s should NOT exist in the namespace %s", currentCertIssuerName, currentCertIssuerNamespace)
				}
			}
			return nil
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	})

	// Verify that the secret used for the default certificate has been removed
	t.It("Validating secret does not exist", func() {
		Eventually(func() error {
			_, err := pkg.GetSecret(currentCertSecretNamespace, currentCertSecretName)
			return err
		}).WithTimeout(waitTimeout).WithPolling(pollingInterval).Should(HaveOccurred())
	})
}

func cleanupTemporaryFiles(files []string) {
	log.Printf("Cleaning up temporary files")
	for _, file := range files {
		_, err := os.Stat(file)
		if os.IsNotExist(err) {
			log.Printf("File %s does not exist", file)
			continue
		}
		err = os.Remove(file)
		Expect(err).ToNot(HaveOccurred())
	}
}
