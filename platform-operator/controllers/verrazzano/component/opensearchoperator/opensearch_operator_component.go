// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"context"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

const (
	// ComponentName is the name of the component
	ComponentName = "opensearch-operator"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoLoggingNamespace

	// ComponentJSONName is the json name of the opensearch-operator component in CRD
	ComponentJSONName = "opensearchOperator"
	// Opster resources
	OpensearchAdminCertificateName                             = "opensearch-admin-cert"
	OpensearchNodeCertificateName                              = "opensearch-node-cert"
	OpensearchClientCertificateName                            = "opensearch-client-cert"
	UsageServerAuth                 certv1.KeyUsage            = "server auth"
	UsageClientAuth                 certv1.KeyUsage            = "client auth"
	PrivateKeyAlgorithm             certv1.PrivateKeyAlgorithm = "RSA"
	PrivateKeyEncoding              certv1.PrivateKeyEncoding  = "PKCS8"
	OpensearchCertCommonName                                   = "admin"
	OpensearchClusterName                                      = "my-opensearch-cluster-2"
)

var OpensearchAdminDNSNames = []string{"admin"}

type opensearchOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return opensearchOperatorComponent{
		HelmComponent: helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			Dependencies:              []string{certmanager.ComponentName},
			AvailabilityObjects: &ready.AvailabilityObjects{
				DeploymentNames: getDeploymentList(),
			},
		},
	}
}

// IsEnabled returns true if the component is enabled for install
func (o opensearchOperatorComponent) IsEnabled(effectiveCr runtime.Object) bool {
	return vzcr.IsOpenSearchOperatorEnabled(effectiveCr)
}

// IsReady - component specific ready-check
func (o opensearchOperatorComponent) IsReady(context spi.ComponentContext) bool {
	if o.HelmComponent.IsReady(context) {
		return o.isReady(context)
	}
	return false
}

// PreInstall runs before components are installed
func (o opensearchOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	cli := ctx.Client()
	log := ctx.Log()

	// create namespace
	ns := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}

	ns.Labels["verrazzano.io/namespace"] = ComponentNamespace
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), cli, &ns, func() error {
		return nil
	}); err != nil {
		return log.ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}

	return o.HelmComponent.PreInstall(ctx)
}

// Install OpenSearchOperator install processing
func (o opensearchOperatorComponent) Install(ctx spi.ComponentContext) error {
	if err := o.HelmComponent.Install(ctx); err != nil {
		return err
	}
	return nil
}

func (o opensearchOperatorComponent) PostInstall(compContext spi.ComponentContext) error {
	// If it is a dry-run, do nothing
	if compContext.IsDryRun() {
		compContext.Log().Debug("cert-manager PostInstall dry run")
		return nil
	}
	return o.createOrUpdateCertResources(compContext)
}
func (o opensearchOperatorComponent) createOrUpdateCertResources(compContext spi.ComponentContext) error {
	var issuerErr error
	compContext.Log().Debug("Applying Certificate for open search operator cert")
	clusterIssuer, err := certmanager.GetClusterIssuer()
	if err != nil {
		return err
	}
	adminCertObject := getCertObject(ComponentNamespace, OpensearchAdminCertificateName)
	nodeCertObject := getCertObject(ComponentNamespace, OpensearchNodeCertificateName)
	clientCertObject := getCertObject(ComponentNamespace, OpensearchClientCertificateName)
	if _, issuerErr = controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &adminCertObject, func() error {
		adminCertObject.Spec = createOpsterAdminCertificate(*clusterIssuer)
		return nil
	}); issuerErr != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the Certificate: %v", issuerErr)
	}
	if _, issuerErr = controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &nodeCertObject, func() error {
		nodeCertObject.Spec = createOpsterNodeCertificate(*clusterIssuer)
		return nil
	}); issuerErr != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the Certificater: %v", issuerErr)
	}

	if _, issuerErr = controllerutil.CreateOrUpdate(context.TODO(), compContext.Client(), &clientCertObject, func() error {
		clientCertObject.Spec = createOpsterClientCertificate(*clusterIssuer)
		return nil
	}); issuerErr != nil {
		return compContext.Log().ErrorfNewErr("Failed to create or update the Certificate: %v", issuerErr)
	}

	return nil
}

func getHoursDuration(hours int) *metav1.Duration {
	ti := metav1.Duration{}
	ti.Duration = time.Duration(hours) * time.Hour
	return &ti
}

// createOpsterAdminCertificate Update the status field for each certificate generated by the Verrazzano ClusterIssuer
func createOpsterAdminCertificate(issuer certv1.ClusterIssuer) certv1.CertificateSpec {
	certCertificate := certv1.CertificateSpec{
		Subject:        getCertificateSubject(),
		CommonName:     OpensearchCertCommonName,
		Duration:       getHoursDuration(2160),
		RenewBefore:    getHoursDuration(360),
		DNSNames:       OpensearchAdminDNSNames,
		SecretName:     OpensearchAdminCertificateName,
		SecretTemplate: nil,
		IssuerRef: certmetav1.ObjectReference{
			Name:  issuer.Name,
			Kind:  "ClusterIssuer",
			Group: "cert-manager.io",
		},
		IsCA:   false,
		Usages: []certv1.KeyUsage{UsageServerAuth, UsageClientAuth},
		PrivateKey: &certv1.CertificatePrivateKey{
			Encoding:  PrivateKeyEncoding,
			Algorithm: PrivateKeyAlgorithm,
			Size:      2048,
		}}
	return certCertificate
}

// createOpsterNodeCertificate Update the status field for each certificate generated by the Verrazzano ClusterIssuer
func createOpsterNodeCertificate(issuer certv1.ClusterIssuer) certv1.CertificateSpec {
	certSpec := certv1.CertificateSpec{
		Subject:        getCertificateSubject(),
		CommonName:     OpensearchCertCommonName, //decalare variable
		Duration:       getHoursDuration(2160),
		RenewBefore:    getHoursDuration(360),
		DNSNames:       getNodeDnsNames(),
		SecretName:     OpensearchNodeCertificateName,
		SecretTemplate: nil,
		IssuerRef: certmetav1.ObjectReference{
			Name:  issuer.Name,
			Kind:  issuer.Kind,
			Group: "cert-manager.io",
		},
		IsCA:   false,
		Usages: []certv1.KeyUsage{UsageServerAuth, UsageClientAuth},
		PrivateKey: &certv1.CertificatePrivateKey{
			Encoding:  PrivateKeyEncoding,
			Algorithm: PrivateKeyAlgorithm,
			Size:      2048,
		},
	}
	return certSpec
}

// createOpsterClientCertificate Update the status field for each certificate generated by the Verrazzano ClusterIssuer
func createOpsterClientCertificate(issuer certv1.ClusterIssuer) certv1.CertificateSpec {
	certSpec := certv1.CertificateSpec{
		Subject:        getCertificateSubject(),
		CommonName:     OpensearchCertCommonName,
		Duration:       getHoursDuration(2160),
		RenewBefore:    getHoursDuration(360),
		DNSNames:       OpensearchAdminDNSNames,
		SecretName:     OpensearchClientCertificateName,
		SecretTemplate: nil,
		IssuerRef: certmetav1.ObjectReference{
			Name:  issuer.Name,
			Kind:  issuer.Kind,
			Group: "cert-manager.io",
		},
		IsCA:   false,
		Usages: []certv1.KeyUsage{UsageServerAuth, UsageClientAuth},
		PrivateKey: &certv1.CertificatePrivateKey{
			Encoding:  PrivateKeyEncoding,
			Algorithm: PrivateKeyAlgorithm,
			Size:      2048,
		},
	}
	return certSpec
}

func getCertObject(namespace, name string) certv1.Certificate {
	return certv1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func getCertificateSubject() *certv1.X509Subject {
	certificateSubject := certv1.X509Subject{
		Organizations: []string{"verrazzano"},
	}
	return &certificateSubject
}

func getNodeDnsNames() []string {
	//my-first-cluster-2
	var dnsList []string
	//dnsList[0] = OpensearchClusterName + "." + constants.VerrazzanoLoggingNamespace + ".svc.cluster.local"
	dnsList[0] = OpensearchClusterName
	return dnsList
}
