// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanager

import (
	"bytes"
	"context"
	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/bom"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	cainjectorDeploymentName = "cert-manager-cainjector"
	webhookDeploymentName = "cert-manager-webhook"

	defaultCAClusterResourceName string = "cert-manager"
	defaultCASecretName          string = "verrazzano-ca-certificate-secret"
)

const custerIssuerTemplate = `
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

type templateData struct {
	Email string
	Server string
	SecretName string
	OCIZoneName string
}

// namespace group, version, resource
var nsGvr = schema.GroupVersionResource{
	Group:    "",
	Version:  "v1",
	Resource: "namespaces",
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

type bashFuncSig func(inArgs ...string) (string, string, error)

var bashFunc bashFuncSig = vzos.RunBash

func setBashFunc(f bashFuncSig) {
	bashFunc = f
}

// Certificate configuration
type Certificate struct {
	CA         *installv1alpha1.CA   `json:"ca,omitempty"`
	ACME       *installv1alpha1.Acme `json:"acme,omitempty"`
}

func (c certManagerComponent) PreInstall(compContext spi.ComponentContext) error {
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

	// Apply the cert-manager manifest
	compContext.Log().Info("Applying cert-manager crds")
	err := c.ApplyManifest(compContext)
	if err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest: %s", err)
		return err
	}
	return nil
}

func (c certManagerComponent) PostInstall(compContext spi.ComponentContext) error {
	compContext.Log().Info("Beginning cert-manager PostInstall")
	certificate := getCertificateConfig(compContext.EffectiveCR())
	if certificate.ACME != nil {
		// Initialize variables for future use
		ociDNSConfigSecret := "verrazzano-container-registry" // compContext.EffectiveCR().Spec.Components.DNS.OCI.OCIConfigSecret
		emailAddress := certificate.ACME.EmailAddress
		ociDNSZoneName := "dns name"

		// Create secret for retrieval
		secret := v1.Secret{}
		if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: ociDNSConfigSecret, Namespace: namespace}, &secret); err != nil {
			compContext.Log().Errorf("Failed to retireve the OCI DNS config secret: %s", err)
			return err
		}

		// Verify the acme environment
		acmeURL := "https://acme-v02.api.letsencrypt.org/directory"
		if acmeEnvironment := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme.Environment; acmeEnvironment != "" && acmeEnvironment != "production" {
			acmeURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
		}

		var buff bytes.Buffer
		clusterIssuerData := templateData{
			Email:       emailAddress,
			Server:      acmeURL,
			SecretName:  ociDNSConfigSecret,
			OCIZoneName: ociDNSZoneName,
		}

		template, err := template.New("clusterIssuer").Parse(custerIssuerTemplate)
		if err != nil {
			compContext.Log().Errorf("Failed to parse the ClusterIssuer yaml template: %s", err)
			return err
		}

		err = template.Execute(&buff, &clusterIssuerData)
		if err != nil {
			compContext.Log().Errorf("Failed to execute the ClusterIssuer template: %s", err)
			return err
		}

		ciObject := &unstructured.Unstructured{Object: map[string]interface{}{}}

		if err := yaml.Unmarshal(buff.Bytes(), ciObject); err != nil {
			compContext.Log().Errorf("Unable to unmarshal yaml: %s", err)
			return err
		}

		compContext.Log().Info("Applying ClusterIssuer with OCI DNS")
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), ciObject, func() error {
			return nil
		}); err != nil {
			compContext.Log().Errorf("Failed to create or update the ClusterIssuer: %s", err)
			return err
		}

	} else if certificate.CA != nil {
		compContext.Log().Info("Applying Issuer for CA cert")
		issuer := certv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "verrazzano-selfsigned-issuer",
				Namespace: certificate.CA.ClusterResourceNamespace,
			},
			Spec:       certv1.IssuerSpec{
				IssuerConfig: certv1.IssuerConfig{
					SelfSigned: &certv1.SelfSignedIssuer{},
				},
			},
			Status:     certv1.IssuerStatus{},
		}
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &issuer, func() error {
			return nil
		}); err != nil {
			compContext.Log().Errorf("Failed to create or update the Issuer: %s", err)
			return err
		}

		compContext.Log().Info("Applying Certificate for CA cert")
		certObject := certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:  "verrazzano-ca-certificate",
				Namespace: certificate.CA.ClusterResourceNamespace,
			},
			Spec: certv1.CertificateSpec{
				SecretName: certificate.CA.SecretName,
				CommonName: "verrazzano-root-ca",
				IsCA: true,
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

		compContext.Log().Info("Applying ClusterIssuer with OCI DNS")
		clusterIssuer := certv1.ClusterIssuer{
			ObjectMeta: metav1.ObjectMeta{
				Name: "verrazzano-cluster-issuer",
			},
			Spec: certv1.IssuerSpec{
				IssuerConfig: certv1.IssuerConfig{
					CA: &certv1.CAIssuer{
						SecretName: certificate.CA.SecretName,
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
	}
	return nil
}

// PatchManifest uses the patch file to patch the cert manager manifest
func (c certManagerComponent) ApplyManifest(compContext spi.ComponentContext) error {
	script := filepath.Join(config.GetInstallDir(), "apply-cert-manager-manifest.sh")

	if compContext.EffectiveCR().Spec.Components.DNS != nil && compContext.EffectiveCR().Spec.Components.DNS.OCI != nil {
		compContext.Log().Info("Patch cert-manager crds to use OCI DNS")
		err := os.Setenv("DNS_TYPE", "oci")
		if err != nil {
			compContext.Log().Errorf("Could not set DNS_TYPE environment variable: %s", err)
			return err
		}
	}
	if _, stderr, err := bashFunc(script); err != nil {
		compContext.Log().Errorf("Failed to apply the cert-manager manifest %s: %s", err, stderr)
		return err
	}
	return nil
}

// IsCertManagerEnabled returns true if the cert-manager is enabled, which is the default
func IsCertManagerEnabled(compContext spi.ComponentContext) bool {
	comp := compContext.EffectiveCR().Spec.Components.CertManager
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// AppendOverrides Build the set of cert-manager overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	certmanager := compContext.EffectiveCR().Spec.Components.CertManager
	if certmanager != nil && (certmanager.Certificate.CA != installv1alpha1.CA{}) {
		kvs = append(kvs, bom.KeyValue{Key: "clusterResourceNamespace", Value: namespace})
	}
	return kvs, nil
}

func IsReady(context spi.ComponentContext, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: certManagerDeploymentName, Namespace: namespace},
		{Name: cainjectorDeploymentName, Namespace: namespace},
		{Name: webhookDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1)
}

func getCertificateConfig(vz *installv1alpha1.Verrazzano) Certificate {
	if vz.Spec.Components.CertManager != nil && (vz.Spec.Components.CertManager.Certificate.Acme != installv1alpha1.Acme{}) {
		return Certificate{
			ACME: &installv1alpha1.Acme{
				Provider:     vz.Spec.Components.CertManager.Certificate.Acme.Provider,
				EmailAddress: vz.Spec.Components.CertManager.Certificate.Acme.EmailAddress,
				Environment:  vz.Spec.Components.CertManager.Certificate.Acme.Environment,
			},
		}
	}
	return Certificate{
		CA: &installv1alpha1.CA{
			ClusterResourceNamespace: getCAClusterResourceNamespace(vz.Spec.Components.CertManager),
			SecretName:               getCASecretName(vz.Spec.Components.CertManager),
		},
	}
}

// getCAClusterResourceNamespace returns the cluster resource name for a CA certificate
func getCAClusterResourceNamespace(cmConfig *installv1alpha1.CertManagerComponent) string {
	if cmConfig == nil {
		return defaultCAClusterResourceName
	}
	ca := cmConfig.Certificate.CA
	// Use default value if not specified
	if ca.ClusterResourceNamespace == "" {
		return defaultCAClusterResourceName
	}
	return ca.ClusterResourceNamespace
}

// getCASecretName returns the secret name for a CA certificate
func getCASecretName(cmConfig *installv1alpha1.CertManagerComponent) string {
	if cmConfig == nil {
		return defaultCASecretName
	}
	ca := cmConfig.Certificate.CA
	// Use default value if not specified
	if ca.SecretName == "" {
		return defaultCASecretName
	}

	return ca.SecretName
}