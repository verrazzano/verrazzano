// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	ctrlerrrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	namespace = "cert-manager"

	certManagerDeploymentName = "cert-manager"
	cainjectorDeploymentName  = "cert-manager-cainjector"
	webhookDeploymentName     = "cert-manager-webhook"

	letsEncryptProd  = "https://acme-v02.api.letsencrypt.org/directory"
	letsEncryptStage = "https://acme-staging-v02.api.letsencrypt.org/directory"

	caSelfSignedIssuerName = "verrazzano-selfsigned-issuer"
	caCertificateName      = "verrazzano-ca-certificate"
	caCertCommonName       = "verrazzano-root-ca"
	caClusterIssuerName    = "verrazzano-cluster-issuer"

	crdDirectory  = "/cert-manager/"
	crdInputFile  = "cert-manager.crds.yaml"
	crdOutputFile = "output.crd.yaml"
)

// Template for ClusterIssuer for Acme certificates
const clusterIssuerTemplate = `
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: verrazzano-cluster-issuer
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
	Email       string
	Server      string
	SecretName  string
	OCIZoneName string
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

// Define bash function here for testing purposes
type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

// PreInstall runs before cert-manager components are installed
// The cert-manager namespace is created
// The cert-manager manifest is patched if needed and applied to create necessary CRDs
func (c certManagerComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Infof("cert-manager PreInstall dry run")
		return nil
	}

	// create cert-manager namespace
	compContext.Log().Info("Adding label needed by network policies to cert-manager namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		return nil
	}); err != nil {
		return ctrlerrrors.RetryableError{
			Source: c.Name(),
			Cause:  fmt.Errorf("Failed to create or update the cert-manager namespace: %s", err),
		}
	}

	// Apply the cert-manager manifest, patching if needed
	compContext.Log().Info("Applying cert-manager crds")
	err := c.applyManifest(compContext)
	if err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest: %s", err)
		return ctrlerrrors.RetryableError{
			Source: c.Name(),
			Cause:  fmt.Errorf("failed to apply the cert-manager manifest: %s", err),
		}
	}
	return nil
}

// PostInstall applies necessary cert-manager resources after the install has occurred
// In the case of an Acme cert, we install Acme resources
// In the case of a CA cert, we install CA resources
func (c certManagerComponent) PostInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Infof("cert-manager PostInstall dry run")
		return nil
	}

	isCAValue, err := isCA(compContext)
	if err != nil {
		compContext.Log().Errorf("Failed to verify the config type: %s", err)
		return ctrlerrrors.RetryableError{Source: c.Name()}
	}
	if !isCAValue {
		// Create resources needed for Acme certificates
		err := createAcmeResources(compContext)
		if err != nil {
			return ctrlerrrors.RetryableError{
				Source: c.Name(),
				Cause:  fmt.Errorf("Failed creating Acme resources: %s", err),
			}
		}
	} else {
		// Create resources needed for CA certificates
		err := createCAResources(compContext)
		if err != nil {
			return ctrlerrrors.RetryableError{
				Source: c.Name(),
				Cause:  fmt.Errorf("Failed creating CA resources: %s", err),
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
		return err
	}

	// Apply the CRD Manifest for CertManager
	if err = k8sutil.NewYAMLApplier(compContext.Client()).ApplyF(outputFile); err != nil {
		return err
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
		compContext.Log().Errorf("Failed to verify the config type: %s", err)
		return []bom.KeyValue{}, ctrlerrrors.RetryableError{Source: ComponentName}
	}
	if isCAValue {
		kvs = append(kvs, bom.KeyValue{Key: "clusterResourceNamespace", Value: namespace})
	}
	return kvs, nil
}

// IsReady checks the state of the expected cert-manager deployments and returns true if they are in a ready state
func (c certManagerComponent) IsReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: certManagerDeploymentName, Namespace: namespace},
		{Name: cainjectorDeploymentName, Namespace: namespace},
		{Name: webhookDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1)
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
	components := compContext.EffectiveCR().Spec.Components
	if components.CertManager == nil {
		return false, errors.New("CertManager object is nil")
	}
	// Check if Ca or Acme is empty
	caNotEmpty := components.CertManager.Certificate.CA != vzapi.CA{}
	acmeNotEmpty := components.CertManager.Certificate.Acme != vzapi.Acme{}
	if caNotEmpty && acmeNotEmpty {
		return false, errors.New("Certificate object Acme and CA cannot be simultaneously populated")
	}
	if caNotEmpty {
		return true, nil
	} else if acmeNotEmpty {
		return false, nil
	} else {
		return false, errors.New("Either Acme or CA certificate authorities must be configured")
	}
}

func isLetsEncryptStaging(compContext spi.ComponentContext) bool {
	acmeEnvironment := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme.Environment
	return acmeEnvironment != "" && acmeEnvironment != "production"
}

// createAcmeResources creates all of the post install resources necessary for cert-manager
func createAcmeResources(compContext spi.ComponentContext) error {
	// Initialize Acme variables for the cluster issuer
	var ociDNSConfigSecret string
	var ociDNSZoneName string
	vzDNS := compContext.EffectiveCR().Spec.Components.DNS
	vzCertAcme := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme
	if vzDNS != nil && vzDNS.OCI != nil {
		ociDNSConfigSecret = vzDNS.OCI.OCIConfigSecret
		ociDNSZoneName = vzDNS.OCI.DNSZoneName
	}
	emailAddress := vzCertAcme.EmailAddress

	// Verify that the secret exists
	secret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: ociDNSConfigSecret, Namespace: namespace}, &secret); err != nil {
		return fmt.Errorf("Failed to retireve the OCI DNS config secret: %s", err)
	}

	// Verify the acme environment and set the server
	acmeServer := letsEncryptProd
	if isLetsEncryptStaging(compContext) {
		acmeServer = letsEncryptStage
	}

	// Create the buffer and the cluster issuer data struct
	var buff bytes.Buffer
	clusterIssuerData := templateData{
		Email:       emailAddress,
		Server:      acmeServer,
		SecretName:  ociDNSConfigSecret,
		OCIZoneName: ociDNSZoneName,
	}

	// Parse the template string and create the template object
	template, err := template.New("clusterIssuer").Parse(clusterIssuerTemplate)
	if err != nil {
		return fmt.Errorf("Failed to parse the ClusterIssuer yaml template: %s", err)
	}

	// Execute the template object with the given data
	err = template.Execute(&buff, &clusterIssuerData)
	if err != nil {
		return fmt.Errorf("Failed to execute the ClusterIssuer template: %s", err)
	}

	// Create an unstructured object from the template output
	ciObject := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err := yaml.Unmarshal(buff.Bytes(), ciObject); err != nil {
		return fmt.Errorf("Unable to unmarshal yaml: %s", err)
	}

	// Update or create the unstructured object
	compContext.Log().Debug("Applying ClusterIssuer with OCI DNS")
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), ciObject, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to create or update the ClusterIssuer: %s", err)
	}
	return nil
}

func createCAResources(compContext spi.ComponentContext) error {
	// Create the issuer resource for CA certs
	compContext.Log().Debug("Applying Issuer for CA cert")
	vzCertCA := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.CA
	issuer := certv1.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caSelfSignedIssuerName,
			Namespace: vzCertCA.ClusterResourceNamespace,
		},
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				SelfSigned: &certv1.SelfSignedIssuer{},
			},
		},
		Status: certv1.IssuerStatus{},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &issuer, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to create or update the Issuer: %s", err)
	}

	// Create the certificate resource for CA cert
	compContext.Log().Debug("Applying Certificate for CA cert")
	certObject := certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      caCertificateName,
			Namespace: vzCertCA.ClusterResourceNamespace,
		},
		Spec: certv1.CertificateSpec{
			SecretName: vzCertCA.SecretName,
			CommonName: caCertCommonName,
			IsCA:       true,
			IssuerRef: certmetav1.ObjectReference{
				Name: issuer.Name,
				Kind: issuer.Kind,
			},
		},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &certObject, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to create or update the Certificate: %s", err)
	}

	// Create the cluster issuer resource for CA cert
	compContext.Log().Debug("Applying ClusterIssuer with OCI DNS")
	clusterIssuer := certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: caClusterIssuerName,
		},
		Spec: certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: vzCertCA.SecretName,
				},
			},
		},
	}
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &clusterIssuer, func() error {
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to create or update the ClusterIssuer: %s", err)
	}
	return nil
}
