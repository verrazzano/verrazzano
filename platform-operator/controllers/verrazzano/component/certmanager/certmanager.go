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
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	certv1client "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzresource "github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
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
	clusterResourceNamespaceKey = "clusterResourceNamespace"

	longestSystemURLPrefix = "elasticsearch.vmi.system"
	preOccupiedspace       = len(longestSystemURLPrefix) + 2

	crdDirectory  = "/cert-manager/"
	crdInputFile  = "cert-manager.crds.yaml"
	crdOutputFile = "output.crd.yaml"

	// ACME-related constants
	defaultCACertificateSecretName = "verrazzano-ca-certificate-secret" //nolint:gosec //#gosec G101
	caAcmeSecretName               = "verrazzano-cert-acme-secret"      //nolint:gosec //#gosec G101

	// Valid Let's Encrypt environment values
	letsencryptProduction    = "production"
	letsEncryptStaging       = "staging"
	letsEncryptProdEndpoint  = "https://acme-v02.api.letsencrypt.org/directory"
	letsEncryptStageEndpoint = "https://acme-staging-v02.api.letsencrypt.org/directory"

	certRequestNameAnnotation = "cert-manager.io/certificate-name"

	// InstancePrincipal is used for instance principle auth type
	instancePrincipal authenticationType = "instance_principal"

	extraArgsKey  = "extraArgs[0]"
	acmeSolverArg = "--acme-http01-solver-image="

	// Uninstall resources
	controllerConfigMap = "cert-manager-controller"
	caInjectorConfigMap = "cert-manager-cainjector-leader-election"

	vzNsLabel = "verrazzano.io/namespace"
)

type authenticationType string

// OCI DNS Secret Auth
type authData struct {
	Region      string             `json:"region"`
	Tenancy     string             `json:"tenancy"`
	User        string             `json:"user"`
	Key         string             `json:"key"`
	Fingerprint string             `json:"fingerprint"`
	AuthType    authenticationType `json:"authtype"`
}

// OCI DNS Secret Auth Wrapper
type ociAuth struct {
	Auth authData `json:"auth"`
}

var (
	// Statically define the well-known Let's Encrypt CA Common Names
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
    preferredChain: ""
    privateKeySecretRef:
      name: {{.AcmeSecretName}}
    solvers:
      - dns01:
          ocidns:
            useInstancePrincipals: {{ .UseInstancePrincipals}}
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
	AcmeSecretName        string
	ClusterIssuerName     string
	Email                 string
	Server                string
	SecretName            string
	OCIZoneName           string
	UseInstancePrincipals bool
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

type getCoreV1ClientFuncType func(log ...vzlog.VerrazzanoLogger) (corev1.CoreV1Interface, error)

var getClientFunc getCoreV1ClientFuncType = k8sutil.GetCoreV1Client

type getCertManagerClientFuncType func() (certv1client.CertmanagerV1Interface, error)

var getCMClientFunc getCertManagerClientFuncType = GetCertManagerClientset

// GetCertManagerClientset Get a CertManager clientset object
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
	// Obtain the CA Common Name for comparison
	comp := vzapi.ConvertCertManagerToV1Beta1(compContext.EffectiveCR().Spec.Components.CertManager)
	issuerCNs, err := findIssuerCommonName(comp.Certificate, isCAConfig)
	if err != nil {
		return err
	}
	// Compare the Issuer CN of all leaf certs in the system with the currently configured Issuer CN
	if err := updateCerts(ctx, log, cmClient, issuerCNs, certList); err != nil {
		return err
	}
	return nil
}

// updateCerts Loop through the certs, and issue a renew request if necessary
func updateCerts(ctx context.Context, log vzlog.VerrazzanoLogger, cmClient certv1client.CertmanagerV1Interface, issuerCNs []string, certList certv1.CertificateList) error {
	for index, currentCert := range certList.Items {
		if currentCert.Name == caCertificateName {
			log.Oncef("Skip renewal of CA certificate")
			continue
		}
		if currentCert.Spec.IssuerRef.Name != verrazzanoClusterIssuerName {
			log.Oncef("Certificate %s/%s not issued by the Verrazzano cluster issuer, skipping", currentCert.Namespace, currentCert.Name)
			continue
		}
		// Get the common name from the cert and update if it doesn't match the issuer CN
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

// getCertIssuerCommonName Gets the CN of the current issuer from the specified Cert secret
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

// renewCertificate Requests a new certificate by updating the status of the Certificate object to "Issuing"
func renewCertificate(ctx context.Context, cmclientv1 certv1client.CertmanagerV1Interface, log vzlog.VerrazzanoLogger, updateCert *certv1.Certificate) error {
	// Update the certificate status to start a renewal; avoid using controllerruntime.CreateOrUpdate(), while
	// it should only do an update we don't want to accidentally create a updateCert
	log.Oncef("Updating certificate %s/%s", updateCert.Namespace, updateCert.Name)

	// If there are any failed certificate requests, they will block a renewal attempt; delete those
	if err := cleanupFailedCertificateRequests(ctx, cmclientv1, log, updateCert); err != nil {
		return err
	}

	// Set the certificate Issuing condition type to True, per guidance by the CertManager team
	cmutil.SetCertificateCondition(updateCert, updateCert.Generation, certv1.CertificateConditionIssuing, certmetav1.ConditionTrue,
		"VerrazzanoUpdate", "Re-issue updated Verrazzano certificates from new ClusterIssuer")
	// Updating the status field only works using the UpdateStatus call via the CertManager typed client interface
	if _, err := cmclientv1.Certificates(updateCert.Namespace).UpdateStatus(ctx, updateCert, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return nil
}

// cleanupFailedCertificateRequests Delete any failed certificate requests associated with a certificate
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
	// use image value for arg override
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, compContext.Log().ErrorNewErr("Failed to get the BOM file for the cert-manager image overrides: ", err)
	}

	images, err := bomFile.BuildImageOverrides("cert-manager")
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if image.Key == extraArgsKey {
			kvs = append(kvs, bom.KeyValue{Key: extraArgsKey, Value: acmeSolverArg + image.Value})
		}
	}

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
	return ready.DeploymentsAreReady(context.Log(), context.Client(), deployments, 1, prefix)
}

// writeCRD writes out CertManager CRD manifests with OCI DNS specifications added
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

// createSnippetWithPadding left pads the OCI DNS snippet with a fixed amount of padding
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
	comp := vzapi.ConvertCertManagerToV1Beta1(compContext.EffectiveCR().Spec.Components.CertManager)
	return validateConfiguration(comp)
}

// validateConfiguration Checks if the configuration is valid and is a CA configuration
// - returns true if it is a CA configuration, false if not
// - returns an error if both CA and ACME settings are configured
func validateConfiguration(comp *v1beta1.CertManagerComponent) (isCA bool, err error) {
	if comp == nil {
		// Is default CA configuration
		return true, nil
	}
	// Check if Ca or Acme is empty
	caNotEmpty := comp.Certificate.CA != v1beta1.CA{}
	acmeNotEmpty := comp.Certificate.Acme != v1beta1.Acme{}
	if caNotEmpty && acmeNotEmpty {
		return false, errors.New("Certificate object Acme and CA cannot be simultaneously populated")
	}
	if caNotEmpty {
		if err := validateCAConfiguration(comp.Certificate.CA); err != nil {
			return true, err
		}
		return true, nil
	} else if acmeNotEmpty {
		if err := validateAcmeConfiguration(comp.Certificate.Acme); err != nil {
			return false, err
		}
		return false, nil
	}
	return false, errors.New("Either Acme or CA certificate authorities must be configured")
}

func validateCAConfiguration(ca v1beta1.CA) error {
	if ca.SecretName == constants.DefaultVerrazzanoCASecretName && ca.ClusterResourceNamespace == ComponentNamespace {
		// if it's the default self-signed config the secret won't exist until created by CertManager
		return nil
	}
	// Otherwise validate the config exists
	_, err := getCASecret(ca)
	return err
}

func getCASecret(ca v1beta1.CA) (*v1.Secret, error) {
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

// validateAcmeConfiguration Validate the ACME/LetsEncrypt values
func validateAcmeConfiguration(acme v1beta1.Acme) error {
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

func isLetsEncryptProvider(acme v1beta1.Acme) bool {
	return strings.ToLower(string(acme.Provider)) == strings.ToLower(string(vzapi.LetsEncrypt))
}

func isLetsEncryptStagingEnv(acme v1beta1.Acme) bool {
	return strings.ToLower(acme.Environment) == letsEncryptStaging
}

func isLetsEncryptProductionEnv(acme v1beta1.Acme) bool {
	return strings.ToLower(acme.Environment) == letsencryptProduction
}

func isLetsEncryptStaging(compContext spi.ComponentContext) bool {
	acmeEnvironment := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme.Environment
	return acmeEnvironment != "" && strings.ToLower(acmeEnvironment) != "production"
}

// createOrUpdateAcmeResources Create or update the ACME ClusterIssuer
// - returns OperationResultNone/error on error
// - returns OperationResultCreated/nil if the CI is created (initial install)
// - returns OperationResultUpdated/nil if the CI is updated
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
	if err := compContext.Client().Get(context.TODO(), crtclient.ObjectKey{Name: ociDNSConfigSecret, Namespace: ComponentNamespace}, &secret); err != nil {
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
		AcmeSecretName:    caAcmeSecretName,
		Email:             emailAddress,
		Server:            acmeServer,
		SecretName:        ociDNSConfigSecret,
		OCIZoneName:       ociDNSZoneName,
	}

	for key := range secret.Data {
		var authProp ociAuth
		if err := yaml.Unmarshal(secret.Data[key], &authProp); err != nil {
			return nil, err
		}
		if authProp.Auth.AuthType == instancePrincipal {
			clusterIssuerData.UseInstancePrincipals = true
			break
		}
	}

	ciObject, err := createAcmeClusterIssuer(compContext.Log(), clusterIssuerData)
	return ciObject, err
}

func createAcmeClusterIssuer(log vzlog.VerrazzanoLogger, clusterIssuerData templateData) (*unstructured.Unstructured, error) {
	var buff bytes.Buffer
	// Parse the template string and create the template object
	issuerTemplate, err := template.New("clusterIssuer").Parse(clusterIssuerTemplate)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to parse the ClusterIssuer yaml template: %v", err)
	}

	// Execute the template object with the given data
	err = issuerTemplate.Execute(&buff, &clusterIssuerData)
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

// createOrUpdateCAResources Create or update the CA ClusterIssuer
// - returns OperationResultNone/error on error
// - returns OperationResultCreated/nil if the CI is created (initial install)
// - returns OperationResultUpdated/nil if the CI is updated
func createOrUpdateCAResources(compContext spi.ComponentContext) (controllerutil.OperationResult, error) {
	vzCertCA := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.CA

	// if the CA cert secret does not exist, create the Issuer and Certificate resources
	secret := v1.Secret{}
	secretKey := crtclient.ObjectKey{Name: vzCertCA.SecretName, Namespace: vzCertCA.ClusterResourceNamespace}
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

func findIssuerCommonName(certificate v1beta1.Certificate, isCAValue bool) ([]string, error) {
	if isCAValue {
		return extractCACommonName(certificate.CA)
	}
	return getACMEIssuerName(certificate.Acme)
}

// getACMEIssuerName Let's encrypt certificates are published, and the intermediate signing CA CNs are well-known
func getACMEIssuerName(acme v1beta1.Acme) ([]string, error) {
	if isLetsEncryptProductionEnv(acme) {
		return letsEncryptProductionCACommonNames, nil
	}
	return letsEncryptStagingCACommonNames, nil
}

func extractCACommonName(ca v1beta1.CA) ([]string, error) {
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

func cleanupUnusedResources(compContext spi.ComponentContext, isCAValue bool) error {
	defaultCANotUsed := func() bool {
		// We're not using the default CA if we're either configured for ACME or it's a Customer-provided CA
		return !isCAValue || compContext.EffectiveCR().Spec.Components.CertManager.Certificate.CA.SecretName != defaultCACertificateSecretName
	}
	client := compContext.Client()
	log := compContext.Log()
	if isCAValue {
		log.Oncef("Clean up ACME issuer secret")
		// clean up ACME secret if present
		if err := deleteObject(client, caAcmeSecretName, ComponentNamespace, &v1.Secret{}); err != nil {
			return err
		}
	}
	if defaultCANotUsed() {
		// Issuer is either the default or Custom issuer; clean up the default Verrazzano issuer resources
		// - self-signed Issuer object
		// - self-signed CA certificate
		// - self-signed secret object
		log.Oncef("Clean up Verrazzano self-signed issuer resources")
		if err := deleteObject(client, caSelfSignedIssuerName, ComponentNamespace, &certv1.Issuer{}); err != nil {
			return err
		}
		if err := deleteObject(client, caCertificateName, ComponentNamespace, &certv1.Certificate{}); err != nil {
			return err
		}
		if err := deleteObject(client, defaultCACertificateSecretName, ComponentNamespace, &v1.Secret{}); err != nil {
			return err
		}
	}
	return nil
}

// uninstallCertManager is the implementation for the cert-manager uninstall step
// this removes cert-manager ConfigMaps from the cluster and after the helm uninstall, deletes the namespace
func uninstallCertManager(compContext spi.ComponentContext) error {
	// Delete the kube-system cert-manager configMaps [controller, caInjector]
	err := vzresource.Resource{
		Name:      controllerConfigMap,
		Namespace: constants.KubeSystem,
		Client:    compContext.Client(),
		Object:    &v1.ConfigMap{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	err = vzresource.Resource{
		Name:      caInjectorConfigMap,
		Namespace: constants.KubeSystem,
		Client:    compContext.Client(),
		Object:    &v1.ConfigMap{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	// Delete the ClusterIssuer created by Verrazzano
	err = vzresource.Resource{
		Name:   verrazzanoClusterIssuerName,
		Client: compContext.Client(),
		Object: &certv1.ClusterIssuer{},
		Log:    compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	// Delete the CA resources if necessary
	err = vzresource.Resource{
		Name:      caSelfSignedIssuerName,
		Namespace: ComponentNamespace,
		Client:    compContext.Client(),
		Object:    &certv1.Issuer{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	err = vzresource.Resource{
		Name:      caCertificateName,
		Namespace: ComponentNamespace,
		Client:    compContext.Client(),
		Object:    &certv1.Certificate{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	err = vzresource.Resource{
		Name:      defaultCACertificateSecretName,
		Namespace: ComponentNamespace,
		Client:    compContext.Client(),
		Object:    &v1.Secret{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	// Delete the ACME secret if present
	err = vzresource.Resource{
		Name:      caAcmeSecretName,
		Namespace: ComponentNamespace,
		Client:    compContext.Client(),
		Object:    &v1.Secret{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	return nil
}

func deleteObject(client crtclient.Client, name string, namespace string, object crtclient.Object) error {
	object.SetName(name)
	object.SetNamespace(namespace)
	if err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, object); err == nil {
		return client.Delete(context.TODO(), object)
	}
	return nil
}

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.CertManager != nil {
			return effectiveCR.Spec.Components.CertManager.ValueOverrides
		}
		return []vzapi.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	if effectiveCR.Spec.Components.CertManager != nil {
		return effectiveCR.Spec.Components.CertManager.ValueOverrides
	}
	return []v1beta1.Overrides{}
}

// validateLongestHostName validates that the longest possible host name for a system endpoint
// is not greater than 64 characters
func validateLongestHostName(effectiveCR runtime.Object) error {
	envName := getEnvironmentName(effectiveCR)
	dnsSuffix, wildcard := getDNSSuffix(effectiveCR)
	spaceOccupied := preOccupiedspace
	longestHostName := fmt.Sprintf("%s.%s.%s", longestSystemURLPrefix, envName, dnsSuffix)
	if len(longestHostName) > 64 {
		if wildcard {
			spaceOccupied = spaceOccupied + len(dnsSuffix)
			return fmt.Errorf("spec.environmentName %s is too long. For the given configuration it must have at most %v characters", envName, 64-spaceOccupied)
		}

		return fmt.Errorf("spec.environmentName %s and DNS suffix %s are too long. For the given configuration they must have at most %v characters in combination", envName, dnsSuffix, 64-spaceOccupied)
	}
	return nil
}

func getEnvironmentName(effectiveCR runtime.Object) string {
	if cr, ok := effectiveCR.(*vzapi.Verrazzano); ok {
		return cr.Spec.EnvironmentName
	}
	cr := effectiveCR.(*v1beta1.Verrazzano)
	return cr.Spec.EnvironmentName
}

func getDNSSuffix(effectiveCR runtime.Object) (string, bool) {
	dnsSuffix, wildcard := "0.0.0.0", true
	if cr, ok := effectiveCR.(*vzapi.Verrazzano); ok {
		if cr.Spec.Components.DNS == nil || cr.Spec.Components.DNS.Wildcard != nil {
			return fmt.Sprintf("%s.%s", dnsSuffix, vzconfig.GetWildcardDomain(cr.Spec.Components.DNS)), wildcard
		} else if cr.Spec.Components.DNS.OCI != nil {
			wildcard = false
			dnsSuffix = cr.Spec.Components.DNS.OCI.DNSZoneName
		} else if cr.Spec.Components.DNS.External != nil {
			wildcard = false
			dnsSuffix = cr.Spec.Components.DNS.External.Suffix
		}
		return dnsSuffix, wildcard
	}

	cr := effectiveCR.(*v1beta1.Verrazzano)
	if cr.Spec.Components.DNS == nil || cr.Spec.Components.DNS.Wildcard != nil {
		return fmt.Sprintf("%s.%s", dnsSuffix, vzconfig.GetWildcardDomain(cr.Spec.Components.DNS)), wildcard
	} else if cr.Spec.Components.DNS.OCI != nil {
		wildcard = false
		dnsSuffix = cr.Spec.Components.DNS.OCI.DNSZoneName
	} else if cr.Spec.Components.DNS.External != nil {
		wildcard = false
		dnsSuffix = cr.Spec.Components.DNS.External.Suffix
	}
	return dnsSuffix, wildcard
}
