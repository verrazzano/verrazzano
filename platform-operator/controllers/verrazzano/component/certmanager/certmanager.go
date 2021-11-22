// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"bytes"
	"context"
	"errors"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
	"text/template"
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
		compContext.Log().Errorf("Failed to create or update the cert-manager namespace: %s", err)
		return err
	}

	// Apply the cert-manager manifest, patching if needed
	compContext.Log().Info("Applying cert-manager crds")
	err := c.ApplyManifest(compContext)
	if err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest: %s", err)
		return err
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
		return err
	}
	if !isCAValue {
		// Create resources needed for Acme certificates
		err := createAcmeResources(compContext)
		if err != nil {
			compContext.Log().Errorf("Failed creating Acme resources: %s", err)
			return err
		}
	} else {
		// Create resources needed for CA certificates
		err := createCAResources(compContext)
		if err != nil {
			compContext.Log().Errorf("Failed creating CA resources: %s", err)
			return err
		}
	}
	return nil
}

// ApplyManifest uses the patch file to patch the cert manager manifest and apply it to the cluster
func (c certManagerComponent) ApplyManifest(compContext spi.ComponentContext) error {
	// find the script location
	script := filepath.Join(config.GetInstallDir(), "apply-cert-manager-manifest.sh")

	// set DNS type to OCI if specified in the effective CR
	if compContext.EffectiveCR().Spec.Components.DNS != nil && compContext.EffectiveCR().Spec.Components.DNS.OCI != nil {
		compContext.Log().Info("Patch cert-manager crds to use OCI DNS")
		err := os.Setenv("DNS_TYPE", "oci")
		if err != nil {
			compContext.Log().Errorf("Could not set DNS_TYPE environment variable: %s", err)
			return err
		}
	}

	// Call and execute script for the given DNS type
	if _, stderr, err := bashFunc(script); err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest %s: %s", err, stderr)
		return err
	}
	return nil
}

// IsEnabled returns true if the cert-manager is enabled, which is the default
func (c certManagerComponent) IsEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// AppendOverrides Build the set of cert-manager overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Verify that we are using CA certs before appending override
	isCAValue, err := isCA(compContext)
	if err != nil {
		compContext.Log().Errorf("Failed to verify the config type: %s", err)
		return []bom.KeyValue{}, err
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
		compContext.Log().Errorf("Failed to retireve the OCI DNS config secret: %s", err)
		return err
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
		compContext.Log().Errorf("Failed to parse the ClusterIssuer yaml template: %s", err)
		return err
	}

	// Execute the template object with the given data
	err = template.Execute(&buff, &clusterIssuerData)
	if err != nil {
		compContext.Log().Errorf("Failed to execute the ClusterIssuer template: %s", err)
		return err
	}

	// Create an unstructured object from the template output
	ciObject := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err := yaml.Unmarshal(buff.Bytes(), ciObject); err != nil {
		compContext.Log().Errorf("Unable to unmarshal yaml: %s", err)
		return err
	}

	// Update or create the unstructured object
	compContext.Log().Debug("Applying ClusterIssuer with OCI DNS")
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), ciObject, func() error {
		return nil
	}); err != nil {
		compContext.Log().Errorf("Failed to create or update the ClusterIssuer: %s", err)
		return err
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
		compContext.Log().Errorf("Failed to create or update the Issuer: %s", err)
		return err
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
		compContext.Log().Errorf("Failed to create or update the Certificate: %s", err)
		return err
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
		compContext.Log().Errorf("Failed to create or update the ClusterIssuer: %s", err)
		return err
	}
	return nil
}
