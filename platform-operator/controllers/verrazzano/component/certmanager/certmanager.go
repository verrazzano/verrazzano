// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"bufio"
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"io"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"net/mail"
	"os"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certv1client "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	certManagerDeploymentName = "cert-manager"
	cainjectorDeploymentName  = "cert-manager-cainjector"
	webhookDeploymentName     = "cert-manager-webhook"

	caSelfSignedIssuerName      = "verrazzano-selfsigned-issuer"
	caCertificateName           = "verrazzano-ca-certificate"
	caCertCommonName            = "verrazzano-root-ca"
	verrazzanoClusterIssuerName = "verrazzano-cluster-issuer"

	crdDirectory  = "/cert-manager/"
	crdInputFile  = "cert-manager.crds.yaml"
	crdOutputFile = "output.crd.yaml"

	clusterResourceNamespaceKey = "clusterResourceNamespace"

	// Valid Let's Encrypt environment values
	letsencryptProduction    = "production"
	letsEncryptStaging       = "staging"
	letsEncryptProdEndpoint  = "https://acme-v02.api.letsencrypt.org/directory"
	letsEncryptStageEndpoint = "https://acme-staging-v02.api.letsencrypt.org/directory"

	certRequestNameAnnotation = "cert-manager.io/certificate-name"
)

var (
	letsEncryptProductionCACommonNames = []string{"R3", "E1", "R4", "E2"}
	letsEncryptStagingCACommonNames    = []string{"(STAGING) Artificial Apricot R3", "(STAGING) Bogus Broccoli X2", "(STAGING) Ersatz Edamame E1"}
)

// Template for ClusterIssuer for looking up Acme certificates for controllerutil.CreateOrUpdate
const clusterIssuerLookupTemplate = `
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: verrazzano-cluster-issuer`

// Template for ClusterIssuer for Acme certificates
const clusterIssuerTemplate = `
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: {{.ClusterIssuerName}}
spec:
  acme:
    email: {{.Email}}
    server: "{{.Server}}"
    privateKeySecretRef:
      name: verrazzano-cert-acme-secret
    solvers:
      - dns01:
          ocidns:
            useInstancePrincipals: false
            serviceAccountSecretRef:
              name: {{.SecretName}}
              key: "oci.yaml"
            ocizonename: {{.OCIZoneName}}`

const snippetSubstring = "rfc2136:\n"

var ociDNSSnippet = strings.Split(`ocidns:
  description:
    ACMEIssuerDNS01ProviderOCIDNS is a structure containing
    the DNS configuration for OCIDNS DNS—Zone Record
    Management API
  properties:
    compartmentid:
      type: string
    ocizonename:
      type: string
    serviceAccountSecretRef:
      properties:
        key:
          description:
            The key of the secret to select from. Must be a
            valid secret key.
          type: string
        name:
          description: Name of the referent.
          type: string
      required:
        - name
      type: object
    useInstancePrincipals:
      type: boolean
  required:
    - ocizonename
  type: object`, "\n")

// Template data for ClusterIssuer
type templateData struct {
	ClusterIssuerName string
	Email             string
	Server            string
	SecretName        string
	OCIZoneName       string
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

// Define bash function here for testing purposes
type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

type getCoreV1ClientFuncType func(log ...vzlog.VerrazzanoLogger) (corev1.CoreV1Interface, error)

var getClientFunc getCoreV1ClientFuncType = k8sutil.GetCoreV1Client

type getCertManagerClientFuncType func() (certv1client.CertmanagerV1Interface, error)

var getCMClientFunc getCertManagerClientFuncType = GetCertManagerClientset

//GetCertManagerClientset Get a CertManager clientset object
func GetCertManagerClientset() (certv1client.CertmanagerV1Interface, error) {
	cfg, err := controllerruntime.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := cmclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return clientset.CertmanagerV1(), nil
}

// checkRenewAllCertificates Update the status field for each certificate generated by the Verrazzano ClusterIssuer
func checkRenewAllCertificates(compContext spi.ComponentContext, isCAConfig bool) error {
	cli := compContext.Client()
	log := compContext.Log()

	certList := certv1.CertificateList{}
	ctx := context.TODO()
	if err := cli.List(ctx, &certList); err != nil {
		return err
	}
	if len(certList.Items) == 0 {
		return nil
	}
	cmClient, err := getCMClientFunc()
	if err != nil {
		return err
	}
	issuerCNs, err := findIssuerCommonName(compContext.EffectiveCR().Spec.Components.CertManager.Certificate, isCAConfig)
	if err != nil {
		return err
	}
	for index, currentCert := range certList.Items {
		if currentCert.Spec.IssuerRef.Name != verrazzanoClusterIssuerName {
			log.Oncef("Certificate %s/%s not issued by the Verrazzano cluster issuer, skipping", currentCert.Namespace, currentCert.Name)
			continue
		}
		certIssuerCN, err := getCertIssuerCommonName(currentCert)
		if err != nil {
			return err
		}
		if !vzstring.SliceContainsString(issuerCNs, certIssuerCN) {
			// If the issuerRef CN is not in the set of configured issuers, we need to renew the existing certs
			if err := renewCertificate(ctx, cmClient, log, &certList.Items[index]); err != nil {
				return err
			}
		}
	}
	return nil
}

func getCertIssuerCommonName(currentCert certv1.Certificate) (string, error) {
	secret, err := getSecret(currentCert.Namespace, currentCert.Spec.SecretName)
	if err != nil {
		return "", err
	}
	certIssuerCN, err := extractCommonNameFromCertSecret(secret)
	if err != nil {
		return "", err
	}
	return certIssuerCN, nil
}

func renewCertificate(ctx context.Context, cmclientv1 certv1client.CertmanagerV1Interface, log vzlog.VerrazzanoLogger, updateCert *certv1.Certificate) error {
	// Update the certificate status to start a renewal; avoid using controllerruntime.CreateOrUpdate(), while
	// it should only do an update we don't want to accidentally create a updateCert
	log.Oncef("Updating certificate %s/%s", updateCert.Namespace, updateCert.Name)

	// If there are any failed certificate requests, they will block a renewal attempt; delete those
	if err := cleanupFailedCertificateRequests(ctx, cmclientv1, log, updateCert); err != nil {
		return err
	}

	// Set the certificate Issuing condition type to True, per guidance by the CertManager team
	cmutil.SetCertificateCondition(updateCert, certv1.CertificateConditionIssuing, certmetav1.ConditionTrue,
		"VerrazzanoUpdate", "Re-issue updated Verrazzano certificates from new ClusterIssuer")
	// Updating the status field only works using the UpdateStatus call via the CertManager typed client interface
	if _, err := cmclientv1.Certificates(updateCert.Namespace).UpdateStatus(ctx, updateCert, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

//cleanupFailedCertificateRequests Delete any failed certificate requests associated with a certificate
func cleanupFailedCertificateRequests(ctx context.Context, cmclientv1 certv1client.CertmanagerV1Interface, log vzlog.VerrazzanoLogger, updateCert *certv1.Certificate) error {
	crNamespaceClient := cmclientv1.CertificateRequests(updateCert.Namespace)
	list, err := crNamespaceClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, cr := range list.Items {
		forCertificate, ok := cr.Annotations[certRequestNameAnnotation]
		if !ok || forCertificate != updateCert.Name {
			log.Debugf("Skipping certificate request %s/s", cr.Namespace, cr.Name)
			continue
		}
		for _, cond := range cr.Status.Conditions {
			if cond.Type == certv1.CertificateRequestConditionReady && cond.Status == certmetav1.ConditionFalse && cond.Reason == certv1.CertificateRequestReasonFailed {
				log.Debugf("Deleting failed certificate request %s/%s", cr.Namespace, cr.Name)
				// certificate request is in a failed state, delete it
				if err := crNamespaceClient.Delete(ctx, cr.Name, metav1.DeleteOptions{}); err != nil {
					log.Errorf("Unable to delete failed certificate request %s/%s: %s", cr.Namespace, cr.Name, err.Error())
					return err
				}
			}
		}
	}
	return nil
}

// applyManifest uses the patch file to patch the cert manager manifest and apply it to the cluster
func (c certManagerComponent) applyManifest(compContext spi.ComponentContext) error {
	crdManifestDir := filepath.Join(config.GetThirdPartyManifestsDir(), crdDirectory)
	// Exclude the input file, since it will be parsed into individual documents
	// Input file containing CertManager CRDs
	inputFile := filepath.Join(crdManifestDir, crdInputFile)
	// Output file format
	outputFile := filepath.Join(crdManifestDir, crdOutputFile)

	// Write out CRD Manifests for CertManager
	err := writeCRD(inputFile, outputFile, isOCIDNS(compContext.EffectiveCR()))
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed writing CRD Manifests for CertManager: %v", err)
	}

	// Apply the CRD Manifest for CertManager
	if err = k8sutil.NewYAMLApplier(compContext.Client(), "").ApplyF(outputFile); err != nil {
		return compContext.Log().ErrorfNewErr("Failed applying CRD Manifests for CertManager: %v", err)

	}

	// Clean up the files written out. This may be different than the files applied
	return cleanTempFiles(outputFile)
}

func cleanTempFiles(tempFiles ...string) error {
	for _, file := range tempFiles {
		if err := os.Remove(file); err != nil {
			return err
		}
	}
	return nil
}

func isOCIDNS(vz *vzapi.Verrazzano) bool {
	return vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil
}

// AppendOverrides Build the set of cert-manager overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Verify that we are using CA certs before appending override
	isCAValue, err := isCA(compContext)
	if err != nil {
		err = compContext.Log().ErrorfNewErr("Failed to verify the config type: %v", err)
		return []bom.KeyValue{}, err
	}
	if isCAValue {
		ns := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.CA.ClusterResourceNamespace
		kvs = append(kvs, bom.KeyValue{Key: clusterResourceNamespaceKey, Value: ns})
	}
	return kvs, nil
}

// isCertManagerReady checks the state of the expected cert-manager deployments and returns true if they are in a ready state
func isCertManagerReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      certManagerDeploymentName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      cainjectorDeploymentName,
			Namespace: ComponentNamespace,
		},
		{
			Name:      webhookDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

//writeCRD writes out CertManager CRD manifests with OCI DNS specifications added
// reads the input CRD file line by line, adding OCI DNS snippets
func writeCRD(inFilePath, outFilePath string, useOCIDNS bool) error {
	infile, err := os.Open(inFilePath)
	if err != nil {
		return err
	}
	defer infile.Close()
	buffer := bytes.Buffer{}
	reader := bufio.NewReader(infile)

	// Flush the current buffer to the filesystem, creating a new manifest file
	flushBuffer := func() error {
		if buffer.Len() < 1 {
			return nil
		}
		outfile, err := os.Create(outFilePath)
		if err != nil {
			return err
		}
		if _, err := outfile.Write(buffer.Bytes()); err != nil {
			return err
		}
		if err := outfile.Close(); err != nil {
			return err
		}
		buffer.Reset()
		return nil
	}

	for {
		// Read the input file line by line
		line, err := reader.ReadBytes('\n')
		if err != nil {
			// If at the end of the file, flush any buffered data
			if err == io.EOF {
				flushErr := flushBuffer()
				return flushErr
			}
			return err
		}
		lineStr := string(line)
		// If the line specifies that the OCI DNS snippet should be written, write it
		if useOCIDNS && strings.HasSuffix(lineStr, snippetSubstring) {
			padding := strings.Repeat(" ", len(strings.TrimSuffix(lineStr, snippetSubstring)))
			snippet := createSnippetWithPadding(padding)
			if _, err := buffer.Write(snippet); err != nil {
				return err
			}
		}
		if _, err := buffer.Write(line); err != nil {
			return err
		}
	}
}

//createSnippetWithPadding left pads the OCI DNS snippet with a fixed amount of padding
func createSnippetWithPadding(padding string) []byte {
	builder := strings.Builder{}
	for _, line := range ociDNSSnippet {
		builder.WriteString(padding)
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return []byte(builder.String())
}

// Check if cert-type is CA, if not it is assumed to be Acme
func isCA(compContext spi.ComponentContext) (bool, error) {
	return validateConfiguration(compContext.EffectiveCR())
}

// validateConfiguration Checks if the configuration is valid and is a CA configuration
// - returns true if it is a CA configuration, false if not
// - returns an error if both CA and ACME settings are configured
func validateConfiguration(cr *vzapi.Verrazzano) (isCA bool, err error) {
	components := cr.Spec.Components
	if components.CertManager == nil {
		// Is default CA configuration
		return true, nil
	}
	// Check if Ca or Acme is empty
	caNotEmpty := components.CertManager.Certificate.CA != vzapi.CA{}
	acmeNotEmpty := components.CertManager.Certificate.Acme != vzapi.Acme{}
	if caNotEmpty && acmeNotEmpty {
		return false, errors.New("Certificate object Acme and CA cannot be simultaneously populated")
	}
	if caNotEmpty {
		if err := validateCAConfiguration(components.CertManager.Certificate.CA); err != nil {
			return true, err
		}
		return true, nil
	} else if acmeNotEmpty {
		if err := validateAcmeConfiguration(components.CertManager.Certificate.Acme); err != nil {
			return false, err
		}
		return false, nil
	}
	return false, errors.New("Either Acme or CA certificate authorities must be configured")
}

func validateCAConfiguration(ca vzapi.CA) error {
	if ca.SecretName == constants.DefaultVerrazzanoCASecretName && ca.ClusterResourceNamespace == ComponentNamespace {
		// if it's the default self-signed config the secret won't exist until created by CertManager
		return nil
	}
	// Otherwise validate the config exists
	_, err := getCASecret(ca)
	return err
}

func getCASecret(ca vzapi.CA) (*v1.Secret, error) {
	name := ca.SecretName
	namespace := ca.ClusterResourceNamespace
	return getSecret(namespace, name)
}

func getSecret(namespace string, name string) (*v1.Secret, error) {
	v1Client, err := getClientFunc()
	if err != nil {
		return nil, err
	}
	return v1Client.Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

//validateAcmeConfiguration Validate the ACME/LetsEncrypt values
func validateAcmeConfiguration(acme vzapi.Acme) error {
	if !isLetsEncryptProvider(acme) {
		return fmt.Errorf("Invalid ACME certificate provider %v", acme.Provider)
	}
	if len(acme.Environment) > 0 && !isLetsEncryptProductionEnv(acme) && !isLetsEncryptStagingEnv(acme) {
		return fmt.Errorf("Invalid Let's Encrypt environment: %s", acme.Environment)
	}
	if _, err := mail.ParseAddress(acme.EmailAddress); err != nil {
		return err
	}
	return nil
}

func isLetsEncryptProvider(acme vzapi.Acme) bool {
	return strings.ToLower(string(acme.Provider)) == strings.ToLower(string(vzapi.LetsEncrypt))
}

func isLetsEncryptStagingEnv(acme vzapi.Acme) bool {
	return strings.ToLower(acme.Environment) == letsEncryptStaging
}

func isLetsEncryptProductionEnv(acme vzapi.Acme) bool {
	return strings.ToLower(acme.Environment) == letsencryptProduction
}

func isLetsEncryptStaging(compContext spi.ComponentContext) bool {
	acmeEnvironment := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme.Environment
	return acmeEnvironment != "" && strings.ToLower(acmeEnvironment) != "production"
}

// createOrUpdateAcmeResources creates all of the post install resources necessary for cert-manager
func createOrUpdateAcmeResources(compContext spi.ComponentContext) (opResult controllerutil.OperationResult, err error) {
	opResult = controllerutil.OperationResultNone
	// Create a lookup object
	getCIObject, err := createAcmeCusterIssuerLookupObject(compContext.Log())
	if err != nil {
		return opResult, err
	}
	// Update or create the unstructured object
	compContext.Log().Debug("Applying ClusterIssuer with OCI DNS")
	if opResult, err = controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), getCIObject, func() error {
		ciObject, err := createACMEIssuerObject(compContext)
		if err != nil {
			return err
		}
		getCIObject.Object["spec"] = ciObject.Object["spec"]
		return nil
	}); err != nil {
		return opResult, compContext.Log().ErrorfNewErr("Failed to create or update the ClusterIssuer: %v", err)
	}
	return opResult, nil
}

func createACMEIssuerObject(compContext spi.ComponentContext) (*unstructured.Unstructured, error) {
	// Initialize Acme variables for the cluster issuer
	var ociDNSConfigSecret string
	var ociDNSZoneName string
	vzDNS := compContext.EffectiveCR().Spec.Components.DNS
	vzCertAcme := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme
	if vzDNS != nil && vzDNS.OCI != nil {
		ociDNSConfigSecret = vzDNS.OCI.OCIConfigSecret
		ociDNSZoneName = vzDNS.OCI.DNSZoneName
	}
	// Verify that the secret exists
	secret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: ociDNSConfigSecret, Namespace: ComponentNamespace}, &secret); err != nil {
		return nil, compContext.Log().ErrorfNewErr("Failed to retrieve the OCI DNS config secret: %v", err)
	}

	emailAddress := vzCertAcme.EmailAddress

	// Verify the acme environment and set the server
	acmeServer := letsEncryptProdEndpoint
	if isLetsEncryptStaging(compContext) {
		acmeServer = letsEncryptStageEndpoint
	}

	// Create the buffer and the cluster issuer data struct
	clusterIssuerData := templateData{
		ClusterIssuerName: verrazzanoClusterIssuerName,
		Email:             emailAddress,
		Server:            acmeServer,
		SecretName:        ociDNSConfigSecret,
		OCIZoneName:       ociDNSZoneName,
	}

	ciObject, err := createAcmeClusterIssuer(compContext.Log(), clusterIssuerData)
	return ciObject, err
}

func createAcmeClusterIssuer(log vzlog.VerrazzanoLogger, clusterIssuerData templateData) (*unstructured.Unstructured, error) {
	var buff bytes.Buffer
	// Parse the template string and create the template object
	template, err := template.New("clusterIssuer").Parse(clusterIssuerTemplate)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to parse the ClusterIssuer yaml template: %v", err)
	}

	// Execute the template object with the given data
	err = template.Execute(&buff, &clusterIssuerData)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to execute the ClusterIssuer template: %v", err)
	}

	// Create an unstructured object from the template output
	ciObject := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err := yaml.Unmarshal(buff.Bytes(), ciObject); err != nil {
		return nil, log.ErrorfNewErr("Failed to unmarshal yaml: %v", err)
	}
	return ciObject, nil
}

func createAcmeCusterIssuerLookupObject(log vzlog.VerrazzanoLogger) (*unstructured.Unstructured, error) {
	ciObject := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err := yaml.Unmarshal([]byte(clusterIssuerLookupTemplate), ciObject); err != nil {
		return nil, log.ErrorfNewErr("Failed to unmarshal yaml: %v", err)
	}
	return ciObject, nil

}

func createOrUpdateCAResources(compContext spi.ComponentContext) (controllerutil.OperationResult, error) {
	vzCertCA := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.CA

	// if the CA cert secret does not exist, create the Issuer and Certificate resources
	secret := v1.Secret{}
	secretKey := client.ObjectKey{Name: vzCertCA.SecretName, Namespace: vzCertCA.ClusterResourceNamespace}
	if err := compContext.Client().Get(context.TODO(), secretKey, &secret); err != nil {
		// Create the issuer resource for CA certs
		compContext.Log().Debug("Applying Issuer for CA cert")
		issuer := certv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caSelfSignedIssuerName,
				Namespace: vzCertCA.ClusterResourceNamespace,
			},
		}
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &issuer, func() error {
			issuer.Spec = certv1.IssuerSpec{
				IssuerConfig: certv1.IssuerConfig{
					SelfSigned: &certv1.SelfSignedIssuer{},
				},
			}
			return nil
		}); err != nil {
			return controllerutil.OperationResultNone,
				compContext.Log().ErrorfNewErr("Failed to create or update the Issuer: %v", err)
		}

		// Create the certificate resource for CA cert
		compContext.Log().Debug("Applying Certificate for CA cert")
		certObject := certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caCertificateName,
				Namespace: vzCertCA.ClusterResourceNamespace,
			},
		}
		commonNameSuffix, err := password.GenerateRandomAlphaLower(8)
		if err != nil {
			return controllerutil.OperationResultNone,
				compContext.Log().ErrorfNewErr("Failed to generate CA common name suffix: %v", err)
		}
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &certObject, func() error {
			certObject.Spec = certv1.CertificateSpec{
				SecretName: vzCertCA.SecretName,
				CommonName: fmt.Sprintf("%s-%s", caCertCommonName, commonNameSuffix),
				IsCA:       true,
				IssuerRef: certmetav1.ObjectReference{
					Name: issuer.Name,
					Kind: issuer.Kind,
				},
			}
			return nil
		}); err != nil {
			return controllerutil.OperationResultNone, compContext.Log().ErrorfNewErr("Failed to create or update the Certificate: %v", err)
		}
	}

	// Create the cluster issuer resource for CA cert
	compContext.Log().Debug("Applying ClusterIssuer")
	clusterIssuer := certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: verrazzanoClusterIssuerName,
		},
	}

	var issuerUpdateErr error
	var opResult controllerutil.OperationResult
	if opResult, issuerUpdateErr = controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &clusterIssuer, func() error {
		clusterIssuer.Spec = certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: vzCertCA.SecretName,
				},
			},
		}
		return nil
	}); issuerUpdateErr != nil {
		return opResult, compContext.Log().ErrorfNewErr("Failed to create or update the ClusterIssuer: %v", issuerUpdateErr)
	}
	return opResult, nil
}

func findIssuerCommonName(certificate vzapi.Certificate, isCAValue bool) ([]string, error) {
	if isCAValue {
		return extractCACommonName(certificate.CA)
	}
	return getACMEIssuerName(certificate.Acme)
}

//getACMEIssuerName Let's encrypt certificates are published, and the intermediate signing CA CNs are well-known
func getACMEIssuerName(acme vzapi.Acme) ([]string, error) {
	if isLetsEncryptProductionEnv(acme) {
		return letsEncryptProductionCACommonNames, nil
	}
	return letsEncryptStagingCACommonNames, nil
}

func extractCACommonName(ca vzapi.CA) ([]string, error) {
	secret, err := getCASecret(ca)
	if err != nil {
		return []string{}, err
	}
	certCN, err := extractCommonNameFromCertSecret(secret)
	return []string{certCN}, err
}

func extractCommonNameFromCertSecret(secret *v1.Secret) (string, error) {
	certBytes, found := secret.Data[v1.TLSCertKey]
	if !found {
		return "", fmt.Errorf("No Certificate data found in secret %s/%s", secret.Namespace, secret.Name)
	}
	leafCACertBytes := []byte{}
	for {
		block, rest := pem.Decode(certBytes)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			// we're only interested in the leaf cert for the CA, as it is the issuing authority in the cluster
			leafCACertBytes = block.Bytes
			break
		}
		certBytes = rest
	}
	cert, err := x509.ParseCertificate(leafCACertBytes)
	if err != nil {
		return "", err
	}
	return cert.Issuer.CommonName, nil
}
