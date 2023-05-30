// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"text/template"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
	acmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	cmclient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	certv1client "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/verrazzano/verrazzano/pkg/constants"
	vzresource "github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/security/password"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	caSelfSignedIssuerName = "verrazzano-selfsigned-issuer"
	caCertificateName      = "verrazzano-ca-certificate"
	caCertCommonName       = "verrazzano-root-ca"

	// LetsEncrypt-related constants

	caAcmeSecretName = "verrazzano-cert-acme-secret" //nolint:gosec //#gosec G101

	// Valid Let's Encrypt environment values
	letsencryptProduction    = "production"
	letsEncryptStaging       = "staging"
	letsEncryptProdEndpoint  = "https://acme-v02.api.letsencrypt.org/directory"
	letsEncryptStageEndpoint = "https://acme-staging-v02.api.letsencrypt.org/directory"

	certRequestNameAnnotation = "cert-manager.io/certificate-name"

	// InstancePrincipal is used for instance principle auth type
	instancePrincipal authenticationType = "instance_principal"
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
          webhook:
            groupName: verrazzano.io
            solverName: oci
            config:
              compartmentOCID: {{ .CompartmentOCID }}
              useInstancePrincipals: {{ .UseInstancePrincipals }}
              ociProfileSecretName: {{.SecretName}}
              ociProfileSecretKey: "oci.yaml"
              ociZoneName: {{.OCIZoneName}}`

// Template data for ClusterIssuer
type templateData struct {
	AcmeSecretName        string
	ClusterIssuerName     string
	Email                 string
	Server                string
	SecretName            string
	OCIZoneName           string
	CompartmentOCID       string
	UseInstancePrincipals bool
}

// CertIssuerType identifies the certificate issuer type
type CertIssuerType string

// var getClientFunc cmcommon.GetCoreV1ClientFuncType = k8sutil.GetCoreV1Client
type getCertManagerClientFuncType func() (certv1client.CertmanagerV1Interface, error)

var getCMClientFunc getCertManagerClientFuncType = GetCertManagerClientset

// GetCertManagerClientset Get a CertManager clientset object
func GetCertManagerClientset() (certv1client.CertmanagerV1Interface, error) {
	cfg, err := k8sutil.GetConfigFromController()
	if err != nil {
		return nil, err
	}
	clientset, err := cmclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return clientset.CertmanagerV1(), nil
}

func UninstallCleanup(log vzlog.VerrazzanoLogger, cli crtclient.Client, namespace string) error {
	log.Debugf("Cleaning up any dangling Cert-Manager resources in namespace %s", namespace)

	if err := deleteResources(log, cli, namespace, &certv1.Issuer{}, createCertManagerGVK("IssuerList")); err != nil {
		return err
	}

	if err := deleteResources(log, cli, namespace, &certv1.CertificateRequest{}, createCertManagerGVK("CertificateRequestList")); err != nil {
		return err
	}

	if err := deleteResources(log, cli, namespace, &certv1.Certificate{}, createCertManagerGVK("CertificateList")); err != nil {
		return err
	}

	if err := deleteResources(log, cli, namespace, &acmev1.Order{}, createAcmeGVK("OrderList")); err != nil {
		return err
	}

	if err := deleteResources(log, cli, namespace, &acmev1.Challenge{}, createAcmeGVK("ChallengeList")); err != nil {
		return err
	}

	return nil
}

func createAcmeGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "acme.cert-manager.io", Version: "v1", Kind: kind}
}

func createCertManagerGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "cert-manager.io", Version: "v1", Kind: kind}
}

func deleteResources(log vzlog.VerrazzanoLogger, cli crtclient.Client, namespace string, obj crtclient.Object, gvk schema.GroupVersionKind) error {
	// Use an unstructured object to get the list of resources
	objectList := &unstructured.UnstructuredList{}
	objectList.SetGroupVersionKind(gvk)
	if err := cli.List(context.TODO(), objectList, crtclient.InNamespace(namespace)); err != nil {
		return err
	}
	for _, item := range objectList.Items {
		itemNamespace := item.GetNamespace()
		itemName := item.GetName()
		log.Progressf("Cleaning up Cert-Manager resource %s in namespace %s", itemName, itemNamespace)
		err := vzresource.Resource{
			Name:      itemName,
			Namespace: itemNamespace,
			Client:    cli,
			Object:    obj,
			Log:       log,
		}.RemoveFinalizersAndDelete()
		if err != nil {
			return err
		}
	}
	return nil
}

// checkRenewAllCertificates Update the status field for each certificate generated by the Verrazzano ClusterIssuer
func checkRenewAllCertificates(log vzlog.VerrazzanoLogger, cli crtclient.Client, config *vzapi.ClusterIssuerComponent) error {
	certList := certv1.CertificateList{}
	ctx := context.TODO()
	if err := cli.List(ctx, &certList); err != nil {
		return err
	}
	if len(certList.Items) == 0 {
		return nil
	}
	// Obtain the CA Common Name for comparison
	issuerCNs, err := findIssuerCommonName(config)
	if err != nil {
		return err
	}
	// Compare the Issuer CN of all leaf certs in the system with the currently configured Issuer CN
	cmClient, err := getCMClientFunc()
	if err != nil {
		return err
	}
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
		if currentCert.Spec.IssuerRef.Name != constants.VerrazzanoClusterIssuerName {
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
	secret, err := cmcommon.GetSecret(currentCert.Namespace, currentCert.Spec.SecretName)
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

// verrazzanoCertManagerResourcesReady Verifies that the Verrazzano ClusterIssuer exists
func (c clusterIssuerComponent) verrazzanoCertManagerResourcesReady(ctx spi.ComponentContext) bool {
	logger := ctx.Log()

	exists, err := vzresource.Resource{
		Name:   constants.VerrazzanoClusterIssuerName,
		Client: ctx.Client(),
		Object: &certv1.ClusterIssuer{},
		Log:    logger,
	}.Exists()
	if err != nil {
		logger.ErrorfThrottled("Error checking for ClusterIssuer %s existence: %v", constants.VerrazzanoClusterIssuerName, err)
	}

	return exists
}

// createOrUpdateAcmeResources Create or update the LetsEncrypt ClusterIssuer
// - returns OperationResultNone/error on error
// - returns OperationResultCreated/nil if the CI is created (initial install)
// - returns OperationResultUpdated/nil if the CI is updated
func createOrUpdateAcmeResources(log vzlog.VerrazzanoLogger, client crtclient.Client, vz *vzapi.Verrazzano, config *vzapi.ClusterIssuerComponent) (opResult controllerutil.OperationResult, err error) {
	opResult = controllerutil.OperationResultNone
	// Create a lookup object
	getCIObject, err := createAcmeCusterIssuerLookupObject(log)
	if err != nil {
		return opResult, err
	}
	// Update or create the unstructured object
	log.Debug("Applying ClusterIssuer with OCI DNS")
	if opResult, err = controllerutil.CreateOrUpdate(context.TODO(), client, getCIObject, func() error {
		ciObject, err := createACMEIssuerObject(log, client, vz, config)
		if err != nil {
			return err
		}
		getCIObject.Object["spec"] = ciObject.Object["spec"]
		return nil
	}); err != nil {
		return opResult, log.ErrorfNewErr("Failed to create or update the ClusterIssuer: %v", err)
	}
	return opResult, nil
}

func createACMEIssuerObject(log vzlog.VerrazzanoLogger, client crtclient.Client, vz *vzapi.Verrazzano, config *vzapi.ClusterIssuerComponent) (*unstructured.Unstructured, error) {
	// Initialize Acme variables for the cluster issuer
	var ociDNSConfigSecret string
	var ociDNSZoneName string
	var ociDNSCompartmentID string
	vzDNS := vz.Spec.Components.DNS
	vzCertAcme := config.LetsEncrypt
	if vzDNS != nil && vzDNS.OCI != nil {
		ociDNSConfigSecret = vzDNS.OCI.OCIConfigSecret
		ociDNSZoneName = vzDNS.OCI.DNSZoneName
		ociDNSCompartmentID = vzDNS.OCI.DNSZoneCompartmentOCID
	}
	// Verify that the secret exists
	secret := v1.Secret{}
	if err := client.Get(context.TODO(), crtclient.ObjectKey{Name: ociDNSConfigSecret, Namespace: config.ClusterResourceNamespace}, &secret); err != nil {
		return nil, log.ErrorfNewErr("Failed to retrieve the OCI DNS config secret: %v", err)
	}

	// Verify the acme environment and set the server
	acmeServer := letsEncryptProdEndpoint
	if cmcommon.IsLetsEncryptStagingEnv(*vzCertAcme) {
		acmeServer = letsEncryptStageEndpoint
	}

	// Create the buffer and the cluster issuer data struct
	clusterIssuerData := templateData{
		ClusterIssuerName: constants.VerrazzanoClusterIssuerName,
		AcmeSecretName:    caAcmeSecretName,
		Email:             vzCertAcme.EmailAddress,
		Server:            acmeServer,
		SecretName:        ociDNSConfigSecret,
		OCIZoneName:       ociDNSZoneName,
		CompartmentOCID:   ociDNSCompartmentID,
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

	ciObject, err := createAcmeClusterIssuer(log, clusterIssuerData)
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
func createOrUpdateCAResources(log vzlog.VerrazzanoLogger, crtClient crtclient.Client, issuerConfig *vzapi.ClusterIssuerComponent) (controllerutil.OperationResult, error) {
	vzCertCA := issuerConfig.CA

	// if the CA cert secret does not exist, create the Issuer and Certificate resources
	secret := v1.Secret{}
	secretKey := crtclient.ObjectKey{Name: vzCertCA.SecretName, Namespace: issuerConfig.ClusterResourceNamespace}
	if err := crtClient.Get(context.TODO(), secretKey, &secret); err != nil {
		// Create the issuer resource for CA certs
		log.Debug("Applying Issuer for CA cert")
		issuer := certv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caSelfSignedIssuerName,
				Namespace: issuerConfig.ClusterResourceNamespace,
			},
		}
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), crtClient, &issuer, func() error {
			issuer.Spec = certv1.IssuerSpec{
				IssuerConfig: certv1.IssuerConfig{
					SelfSigned: &certv1.SelfSignedIssuer{},
				},
			}
			return nil
		}); err != nil {
			return controllerutil.OperationResultNone,
				log.ErrorfThrottledNewErr("Failed to create or update the Issuer: %v", err.Error())
		}

		// Create the certificate resource for CA cert
		log.Debug("Applying Certificate for CA cert")
		certObject := certv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      caCertificateName,
				Namespace: issuerConfig.ClusterResourceNamespace,
			},
		}
		commonNameSuffix, err := password.GenerateRandomAlphaLower(8)
		if err != nil {
			return controllerutil.OperationResultNone,
				log.ErrorfNewErr("Failed to generate CA common name suffix: %v", err)
		}
		if _, err := controllerutil.CreateOrUpdate(context.TODO(), crtClient, &certObject, func() error {
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
			return controllerutil.OperationResultNone, log.ErrorfNewErr("Failed to create or update the Certificate: %v", err)
		}
	}

	// Create the cluster issuer resource for CA cert
	log.Debug("Applying ClusterIssuer")
	clusterIssuer := certv1.ClusterIssuer{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.VerrazzanoClusterIssuerName,
		},
	}

	var issuerUpdateErr error
	var opResult controllerutil.OperationResult
	if opResult, issuerUpdateErr = controllerutil.CreateOrUpdate(context.TODO(), crtClient, &clusterIssuer, func() error {
		clusterIssuer.Spec = certv1.IssuerSpec{
			IssuerConfig: certv1.IssuerConfig{
				CA: &certv1.CAIssuer{
					SecretName: vzCertCA.SecretName,
				},
			},
		}
		return nil
	}); issuerUpdateErr != nil {
		return opResult, log.ErrorfNewErr("Failed to create or update the ClusterIssuer: %v", issuerUpdateErr)
	}
	return opResult, nil
}

func findIssuerCommonName(config *vzapi.ClusterIssuerComponent) ([]string, error) {
	isCAIssuer, err := config.IsCAIssuer()
	if err != nil {
		return []string{}, err
	}
	if isCAIssuer {
		return extractCACommonName(config)
	}
	return getACMEIssuerName(config.LetsEncrypt)
}

// getACMEIssuerName Let's encrypt certificates are published, and the intermediate signing CA CNs are well-known
func getACMEIssuerName(acme *vzapi.LetsEncryptACMEIssuer) ([]string, error) {
	if acme == nil {
		return []string{}, fmt.Errorf("Illegal state, LetsEncrypt issuer not configured")
	}
	if cmcommon.IsLetsEncryptProductionEnv(*acme) {
		return letsEncryptProductionCACommonNames, nil
	}
	return letsEncryptStagingCACommonNames, nil
}

func extractCACommonName(config *vzapi.ClusterIssuerComponent) ([]string, error) {
	secret, err := cmcommon.GetSecret(config.ClusterResourceNamespace, config.CA.SecretName)
	if err != nil {
		return []string{}, err
	}
	certCN, err := extractCommonNameFromCertSecret(secret)
	return []string{certCN}, err
}

func extractCommonNameFromCertSecret(secret *v1.Secret) (string, error) {
	certBytes, found := secret.Data[v1.TLSCertKey]
	if !found {
		return "", fmt.Errorf("no Certificate data found in secret %s/%s", secret.Namespace, secret.Name)
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
	effectiveCR := compContext.EffectiveCR()
	clusterIssuer := effectiveCR.Spec.Components.ClusterIssuer
	if clusterIssuer == nil {
		compContext.Log().ErrorfThrottled("Cluster issuer not defined, skipping cleanup of unused resources")
		return nil
	}
	if err := cleanupUnusedResourcesForNamespace(compContext, clusterIssuer.ClusterResourceNamespace, clusterIssuer, isCAValue); err != nil {
		return err
	}
	// Edge case for Custom CA, the user may change the clusterResourceNamespace as part of the change to a custom CA and
	// leave resources abandoned in the Verrazzano cert-manager namespace; take a second pass and clean those up if
	// our Cert-Manager is in play
	if vzcr.IsCertManagerEnabled(effectiveCR) {
		if err := cleanupUnusedResourcesForNamespace(compContext, constants.CertManagerNamespace, clusterIssuer, isCAValue); err != nil {
			return err
		}
	}
	return nil
}

func cleanupUnusedResourcesForNamespace(compContext spi.ComponentContext, clusterResourceNamespace string, clusterIssuer *vzapi.ClusterIssuerComponent, isCAValue bool) error {
	client := compContext.Client()
	log := compContext.Log()
	defaultCANotUsed := func() bool {
		// We're not using the default CA if we're either configured for ACME or it's a Customer-provided CA
		return !isCAValue || clusterIssuer.CA.SecretName != constants.DefaultVerrazzanoCASecretName
	}
	if isCAValue {
		log.Oncef("Clean up ACME issuer secret")
		// clean up ACME secret if present
		if err := deleteObject(client, caAcmeSecretName, clusterResourceNamespace, &v1.Secret{}); err != nil {
			return err
		}
	}
	if defaultCANotUsed() {
		// Issuer is either the default or Custom issuer; clean up the default Verrazzano issuer resources
		// - self-signed Issuer object
		// - self-signed CA certificate
		// - self-signed secret object
		log.Oncef("Clean up Verrazzano self-signed issuer resources")
		if err := deleteObject(client, caSelfSignedIssuerName, clusterResourceNamespace, &certv1.Issuer{}); err != nil {
			return err
		}
		if err := deleteObject(client, caCertificateName, clusterResourceNamespace, &certv1.Certificate{}); err != nil {
			return err
		}
		if err := deleteObject(client, constants.DefaultVerrazzanoCASecretName, clusterResourceNamespace, &v1.Secret{}); err != nil {
			return err
		}
	}
	return nil
}

func (c clusterIssuerComponent) createOrUpdateClusterIssuer(compContext spi.ComponentContext) error {

	effectiveCR := compContext.EffectiveCR()
	isCAValue, err := vzcr.IsCAConfig(effectiveCR)
	if err != nil {
		return compContext.Log().ErrorfNewErr("Failed to verify the config type: %v", err)
	}

	clusterIssuerConfig := effectiveCR.Spec.Components.ClusterIssuer
	if clusterIssuerConfig == nil {
		return compContext.Log().ErrorfThrottledNewErr("Cluster issuer is not configured")
	}

	var opResult controllerutil.OperationResult
	if !isCAValue {
		// Create resources needed for Acme certificates
		if opResult, err = createOrUpdateAcmeResources(compContext.Log(), compContext.Client(), effectiveCR, clusterIssuerConfig); err != nil {
			return compContext.Log().ErrorfNewErr("Failed creating Acme resources: %v", err)
		}
	} else {
		// Create resources needed for CA certificates
		if opResult, err = createOrUpdateCAResources(compContext.Log(), compContext.Client(), clusterIssuerConfig); err != nil {
			msg := fmt.Sprintf("Failed creating CA resources: %v", err)
			compContext.Log().Once(msg)
			return fmt.Errorf(msg)
		}
	}
	if opResult == controllerutil.OperationResultCreated {
		// We're in the initial install phase, and created the ClusterIssuer for the first time,
		// so skip the renewal checks
		compContext.Log().Oncef("Initial install, skipping certificate renewal checks")
		return nil
	}
	// CertManager configuration was updated, cleanup any old resources from previous configuration
	// and renew certificates against the new ClusterIssuer
	if err := cleanupUnusedResources(compContext, isCAValue); err != nil {
		return err
	}
	if err := checkRenewAllCertificates(compContext.Log(), compContext.Client(), clusterIssuerConfig); err != nil {
		compContext.Log().Errorf("Error requesting certificate renewal: %s", err.Error())
		return err
	}
	return nil
}

// uninstallVerrazzanoCertManagerResources is the implementation for the cert-manager uninstall step
// this removes cert-manager ConfigMaps from the cluster and after the helm uninstall, deletes the namespace
func (c clusterIssuerComponent) uninstallVerrazzanoCertManagerResources(compContext spi.ComponentContext) error {

	issuerConfig := compContext.EffectiveCR().Spec.Components.ClusterIssuer

	if err := c.deleteCertManagerIssuerResources(compContext.Log(), compContext.Client(), issuerConfig); err != nil {
		return err
	}

	err := vzresource.Resource{
		Name:      constants.DefaultVerrazzanoCASecretName,
		Namespace: issuerConfig.ClusterResourceNamespace,
		Client:    compContext.Client(),
		Object:    &v1.Secret{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}

	// Delete the LetsEncrypt secret if present
	err = vzresource.Resource{
		Name:      caAcmeSecretName,
		Namespace: issuerConfig.ClusterResourceNamespace,
		Client:    compContext.Client(),
		Object:    &v1.Secret{},
		Log:       compContext.Log(),
	}.Delete()
	if err != nil {
		return err
	}
	return nil
}

func (c clusterIssuerComponent) deleteCertManagerIssuerResources(log vzlog.VerrazzanoLogger, client crtclient.Client, config *vzapi.ClusterIssuerComponent) error {

	// Delete the ClusterIssuer created by Verrazzano
	err := vzresource.Resource{
		Name:   constants.VerrazzanoClusterIssuerName,
		Client: client,
		Object: &certv1.ClusterIssuer{},
		Log:    log,
	}.Delete()
	if err != nil {
		return err
	}

	// Delete the CA resources if necessary
	err = vzresource.Resource{
		Name:      caSelfSignedIssuerName,
		Namespace: config.ClusterResourceNamespace,
		Client:    client,
		Object:    &certv1.Issuer{},
		Log:       log,
	}.Delete()
	if err != nil {
		return err
	}
	err = vzresource.Resource{
		Name:      caCertificateName,
		Namespace: config.ClusterResourceNamespace,
		Client:    client,
		Object:    &certv1.Certificate{},
		Log:       log,
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
