// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	cmclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	v12 "github.com/jetstack/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"io"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"net/mail"
	"os"
	"path/filepath"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
	"text/template"

	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	cmutil "github.com/jetstack/cert-manager/pkg/api/util"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
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

type getClientFuncType func() (corev1.CoreV1Interface, error)

var getClientFunc getClientFuncType = GetCoreV1Client

func GetCoreV1Client() (corev1.CoreV1Interface, error) {
	goClient, err := k8sutil.GetGoClient()
	if err != nil {
		return nil, err
	}
	return goClient.CoreV1(), nil
}

func (c certManagerComponent) createOrUpdateClusterIssuer(compContext spi.ComponentContext) error {
	isCAValue, err := isCA(compContext)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to verify the config type: %v", err)
	}
	var opResult controllerutil.OperationResult
	if !isCAValue {
		// Create resources needed for Acme certificates
		if opResult, err = createOrUpdateAcmeResources(compContext); err != nil {
			return compContext.Log().ErrorfNewErr("Failed creating Acme resources: %v", err)
		}
	} else {
		// Create resources needed for CA certificates
		if opResult, err = createOrUpdateCAResources(compContext); err != nil {
			return compContext.Log().ErrorfNewErr("Failed creating CA resources: %v", err)
		}
	}
	if opResult == controllerutil.OperationResultUpdated {
		// If the ClusterIssuer has been updated, request a renewal of all system certificates
		compContext.Log().Infof("ClusterIssuer updated, renewing system certificates")
		if err := renewAllSystemCertificates(compContext.Client(), compContext.Log()); err != nil {
			compContext.Log().Errorf("Error requesting certificate renewal: %s", err.Error())
			return err
		}
	}
	return nil
}

// renewAllSystemCertificates Update the status field for each certificate generated by the Verrazzano ClusterIssuer
func renewAllSystemCertificates(cli client.Client, log vzlog.VerrazzanoLogger) error {
	certList := certv1.CertificateList{}
	ctx := context.Background()
	if err := cli.List(ctx, &certList); err != nil {
		return err
	}
	// Updating the status field only works using the UpdateStatus call via the CertManager typed client interface
	cfg := controllerruntime.GetConfigOrDie()
	cmclientv1 := cmclient.NewForConfigOrDie(cfg).CertmanagerV1()
	for index, currentCert := range certList.Items {
		if currentCert.Spec.IssuerRef.Name != verrazzanoClusterIssuerName {
			log.Infof("Certificate %s/%s not issued by the Verrazzano cluster issuer, skipping", currentCert.Namespace, currentCert.Name)
			continue
		}
		//cert :=
		if err := renewCertificate(ctx, cmclientv1, log, &certList.Items[index]); err != nil {
			return err
		}
	}
	return nil
}

func renewCertificate(ctx context.Context, cmclientv1 v12.CertmanagerV1Interface, log vzlog.VerrazzanoLogger, updateCert *certv1.Certificate) error {
	// Update the certificate status to start a renewal; avoid using controllerruntime.CreateOrUpdate(), while
	// it should only do an update we don't want to accidentally create a updateCert
	log.Infof("Updating certificate %s/%s", updateCert.Namespace, updateCert.Name)

	// Set the certificate Issuing condition type to True, per guidance by the CertManager team
	cmutil.SetCertificateCondition(updateCert, certv1.CertificateConditionIssuing, certmetav1.ConditionTrue,
		"VerrazzanoUpdate", "Re-issue updated Verrazzano certificates from new ClusterIssuer")
	// Updating the Status field isn't working with
	if _, err := cmclientv1.Certificates(updateCert.Namespace).UpdateStatus(ctx, updateCert, metav1.UpdateOptions{}); err != nil {
		return err
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
	v1Client, err := getClientFunc()
	if err != nil {
		return err
	}
	_, err = v1Client.Secrets(ca.ClusterResourceNamespace).Get(context.Background(), ca.SecretName, metav1.GetOptions{})
	return err
}

//validateAcmeConfiguration Validate the ACME/LetsEncrypt values
func validateAcmeConfiguration(acme vzapi.Acme) error {
	if strings.ToLower(string(acme.Provider)) != strings.ToLower(string(vzapi.LetsEncrypt)) {
		return fmt.Errorf("Invalid ACME certificate provider %v", acme.Provider)
	}
	if len(acme.Environment) > 0 && strings.ToLower(acme.Environment) != letsencryptProduction && strings.ToLower(acme.Environment) != letsEncryptStaging {
		return fmt.Errorf("Invalid Let's Encrypt environment: %s", acme.Environment)
	}
	if _, err := mail.ParseAddress(acme.EmailAddress); err != nil {
		return err
	}
	return nil
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
