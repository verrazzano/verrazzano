// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certmanagerocidns

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	certManagerDeploymentName = "cert-manager"
	ocidnsDeploymentName      = "cert-manager-ocidns-provider"

	letsEncryptProd  = "https://acme-v02.api.letsencrypt.org/directory"
	letsEncryptStage = "https://acme-staging-v02.api.letsencrypt.org/directory"
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
          webhook:
            groupName: verrazzano.io
            solverName: oci
            config:
              useInstancePrincipals: false
              ociProfileSecretName: {{.SecretName}}
              ociProfileSecretKey: "oci.yaml"
              ociZoneName: {{.OCIZoneName}}`

// Template data for ClusterIssuer
type templateData struct {
	Email       string
	Server      string
	SecretName  string
	OCIZoneName string
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

// PreInstall runs before cert-manager components are installed
// The cert-manager namespace is created
// The cert-manager manifest is patched if needed and applied to create necessary CRDs
func (c certManagerOciDNSComponent) PreInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-ocidns PreInstall dry run")
		return nil
	}
	return nil
}

// PostInstall applies necessary cert-manager resources after the install has occurred
// In the case of an Acme cert, we install Acme resources
// In the case of a CA cert, cert-manager should have installed CA resources
func (c certManagerOciDNSComponent) PostInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager-ocidns PostInstall dry run")
		return nil
	}

	isCAValue, err := certmanager.IsCA(compContext)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to verify the config type: %v", err)
	}
	if !isCAValue {
		// Create resources needed for Acme certificates
		err := createAcmeResources(compContext)
		if err != nil {
			return compContext.Log().ErrorfNewErr("Failed creating Acme resources: %v", err)
		}
	}
	return nil
}

// AppendOverrides Build the set of overrides for the helm install
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	return kvs, nil
}

// isCertManagerReady checks the state of the expected cert-manager deployments and returns true if they are in a ready state
func isCertManagerOciDNSReady(context spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{Name: certManagerDeploymentName, Namespace: ComponentNamespace},
	}
	if certmanager.IsOCIDNS(context.EffectiveCR()) {
		deployments = append(deployments, types.NamespacedName{Name: ocidnsDeploymentName, Namespace: ComponentNamespace})
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1, prefix)
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
	if err := compContext.Client().Get(context.TODO(), client.ObjectKey{Name: ociDNSConfigSecret, Namespace: ComponentNamespace}, &secret); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to retrieve the OCI DNS config secret: %v", err)
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
		return compContext.Log().ErrorfNewErr("Failed to parse the ClusterIssuer yaml template: %v", err)
	}

	// Execute the template object with the given data
	err = template.Execute(&buff, &clusterIssuerData)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to execute the ClusterIssuer template: %v", err)
	}

	// Create an unstructured object from the template output
	ciObject := &unstructured.Unstructured{Object: map[string]interface{}{}}
	if err := yaml.Unmarshal(buff.Bytes(), ciObject); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to unmarshal yaml: %v", err)
	}

	// Update or create the unstructured object
	compContext.Log().Debug("Applying ClusterIssuer with OCI DNS")
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), ciObject, func() error {
		return nil
	}); err != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the ClusterIssuer: %v", err)
	}
	return nil
}
