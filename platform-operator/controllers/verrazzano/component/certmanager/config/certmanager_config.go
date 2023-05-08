// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package config

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	acmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	cmcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"text/template"

	cmutil "github.com/cert-manager/cert-manager/pkg/api/util"
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
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	crtclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

const (
	caSelfSignedIssuerName = "verrazzano-selfsigned-issuer"
	caCertificateName      = "verrazzano-ca-certificate"
	caCertCommonName       = "verrazzano-root-ca"

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
	crdsExist, err := common.CertManagerCrdsExist(cli)
	if err != nil {
		return err
	}
	if !crdsExist {
		return nil
	}

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
func (c certManagerConfigComponent) verrazzanoCertManagerResourcesReady(ctx spi.ComponentContext) bool {
	logger := ctx.Log()

	if !c.cmCRDsExist(ctx.Log(), ctx.Client()) {
		return false
	}

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

func (c certManagerConfigComponent) cmCRDsExist(log vzlog.VerrazzanoLogger, cli crtclient.Client) bool {
	crdsExist, err := common.CertManagerCrdsExist(cli)
	if err != nil {
		log.ErrorfThrottled("Error checking if CertManager CRDs exist: %v", err.Error())
		return false
	}
	return crdsExist
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
	var ociDNSCompartmentID string
	vzDNS := compContext.EffectiveCR().Spec.Components.DNS
	vzCertAcme := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme
	if vzDNS != nil && vzDNS.OCI != nil {
		ociDNSConfigSecret = vzDNS.OCI.OCIConfigSecret
		ociDNSZoneName = vzDNS.OCI.DNSZoneName
		ociDNSCompartmentID = vzDNS.OCI.DNSZoneCompartmentOCID
	}
	// Verify that the secret exists
	secret := v1.Secret{}
	if err := compContext.Client().Get(context.TODO(), crtclient.ObjectKey{Name: ociDNSConfigSecret, Namespace: ComponentNamespace}, &secret); err != nil {
		return nil, compContext.Log().ErrorfNewErr("Failed to retrieve the OCI DNS config secret: %v", err)
	}

	emailAddress := vzCertAcme.EmailAddress

	// Verify the acme environment and set the server
	acmeServer := letsEncryptProdEndpoint
	if cmcommon.IsLetsEncryptStaging(compContext) {
		acmeServer = letsEncryptStageEndpoint
	}

	// Create the buffer and the cluster issuer data struct
	clusterIssuerData := templateData{
		ClusterIssuerName: constants.VerrazzanoClusterIssuerName,
		AcmeSecretName:    caAcmeSecretName,
		Email:             emailAddress,
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
			Name: constants.VerrazzanoClusterIssuerName,
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
	if cmcommon.IsLetsEncryptProductionEnv(acme) {
		return letsEncryptProductionCACommonNames, nil
	}
	return letsEncryptStagingCACommonNames, nil
}

func extractCACommonName(ca v1beta1.CA) ([]string, error) {
	secret, err := cmcommon.GetCASecret(ca)
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

func (c certManagerConfigComponent) createOrUpdateClusterIssuer(compContext spi.ComponentContext) error {
	isCAValue, err := cmcommon.IsCA(compContext)
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
	if err := checkRenewAllCertificates(compContext, isCAValue); err != nil {
		compContext.Log().Errorf("Error requesting certificate renewal: %s", err.Error())
		return err
	}
	return nil
}

// uninstallVerrazzanoCertManagerResources is the implementation for the cert-manager uninstall step
// this removes cert-manager ConfigMaps from the cluster and after the helm uninstall, deletes the namespace
func (c certManagerConfigComponent) uninstallVerrazzanoCertManagerResources(compContext spi.ComponentContext) error {

	if err := c.deleteCertManagerIssuerResources(compContext); err != nil {
		return err
	}

	err := vzresource.Resource{
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

func (c certManagerConfigComponent) deleteCertManagerIssuerResources(compContext spi.ComponentContext) error {

	// Delete the ClusterIssuer created by Verrazzano
	if !c.cmCRDsExist(compContext.Log(), compContext.Client()) {
		compContext.Log().Progressf("CertManager CRDs do not exist, skipping ClusterIssuer cleanup")
		return nil
	}

	err := vzresource.Resource{
		Name:   constants.VerrazzanoClusterIssuerName,
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
