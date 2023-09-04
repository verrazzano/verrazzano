// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dex

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"text/template"

	"github.com/google/uuid"
	"github.com/verrazzano/verrazzano/pkg/bom"
	v8oconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzpassword "github.com/verrazzano/verrazzano/pkg/security/password"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var writeFileFunc = os.WriteFile

const (
	configIssuer    = "config.issuer"
	ingressClassKey = "ingress.className"
	dnsTarget       = "dnsTarget"
	hostsHost       = "host"
	tlsHosts        = "tlsHosts"
	tlsSecret       = "dexSecret"
	dexTLSSecret    = "dex-tls" //nolint:gosec //#gosec G101
	pkceClient      = "verrazzano-pkce"
	pgClient        = "verrazzano-pg"

	httpPrefix      = "http://"
	dexClientSecret = "clientSecret"

	// ES secret keys
	adminUsernameKey = "username"
	adminPasswordKey = "password"

	dexCertificateName = "dex-tls" //nolint:gosec //#gosec G101
	helmValuesFile     = "dex-values.yaml"

	tmpFilePrefix       = "dex-overrides-"
	tmpSuffix           = "yaml"
	tmpFileCleanPattern = tmpFilePrefix + ".*\\." + tmpSuffix
)

type userData struct {
	Email    string
	Hash     string
	UserName string
	UserID   string
}

type clientData struct {
	ClientID     string
	RedirectURIs string
	ClientName   string
	ClientSecret string
}

type redirectURIsData struct {
	DNSSubDomain string
	OSHostExists bool
}

const pkceClientUrisTemplate = `redirectURIs: [
      "https://verrazzano.{{.DNSSubDomain}}/*",
      "https://verrazzano.{{.DNSSubDomain}}/_authentication_callback",
      "https://opensearch.vmi.system.{{.DNSSubDomain}}/*",
      "https://opensearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://prometheus.vmi.system.{{.DNSSubDomain}}/*",
      "https://prometheus.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://grafana.vmi.system.{{.DNSSubDomain}}/*",
      "https://grafana.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://osd.vmi.system.{{.DNSSubDomain}}/*",
      "https://osd.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://kiali.vmi.system.{{.DNSSubDomain}}/*",
      "https://kiali.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://thanos-query-store.{{.DNSSubDomain}}/*",
      "https://thanos-query-store.{{.DNSSubDomain}}/_authentication_callback",
      "https://opensearch.logging.{{.DNSSubDomain}}/_authentication_callback",
      "https://opensearch.logging.{{.DNSSubDomain}}/*",
      "https://osd.logging.{{.DNSSubDomain}}/*",
      "https://osd.logging.{{.DNSSubDomain}}/_authentication_callback",
      "https://thanos-query.{{.DNSSubDomain}}/*",
      "https://thanos-query.{{.DNSSubDomain}}/_authentication_callback",
      "https://thanos-ruler.{{.DNSSubDomain}}/*",
      "https://thanos-ruler.{{.DNSSubDomain}}/_authentication_callback",
      "https://jaeger.{{.DNSSubDomain}}/*",
      "https://alertmanager.{{.DNSSubDomain}}/*",
      "https://alertmanager.{{.DNSSubDomain}}/_authentication_callback"{{ if .OSHostExists}},
      "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/*",
      "https://elasticsearch.vmi.system.{{.DNSSubDomain}}/_authentication_callback",
      "https://kibana.vmi.system.{{.DNSSubDomain}}/*",
      "https://kibana.vmi.system.{{.DNSSubDomain}}/_authentication_callback"{{end}}
    ]`

const staticClientTemplate = `config:
  staticClients:
`

const clientTemplate = `  - id: "{{.ClientID}}"
    name: "{{.ClientName}}"
    secret: {{.ClientSecret}}
    {{.RedirectURIs}}
`

//nolint:gosec //#gosec G101
const staticPasswordTemplate = `config:
  staticPasswords:
`

//nolint:gosec //#gosec G101
const passwordTemplate = `  - email: "{{.Email}}"
    hash: "{{.Hash}}"
    username: "{{.UserName}}"
    userID: "{{.UserID}}"
`

// GetOverrides gets the installation overrides for the Dex component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Dex != nil {
			return effectiveCR.Spec.Components.Dex.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*v1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.Dex != nil {
			return effectiveCR.Spec.Components.Dex.ValueOverrides
		}
		return []v1beta1.Overrides{}
	}
	return []vzapi.Overrides{}
}

// AppendDexOverrides appends the default overrides for the Dex component
func AppendDexOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, err
	}

	image, err := bomFile.BuildImageOverrides(ComponentName)
	if err != nil {
		return kvs, ctx.Log().ErrorfNewErr("Failed to build Dex image overrides from the Verrazzano BOM: %v", err)
	}
	kvs = append(kvs, image...)
	ctx.Log().Infof("AppendDexOverrides: Dex image %s", image)

	// Get DNS Domain Configuration
	dnsSubDomain, err := getDNSDomain(ctx.Client(), ctx.EffectiveCR())
	if err != nil {
		ctx.Log().Errorf("Component Dex failed retrieving DNS sub domain: %v", err)
		return nil, err
	}
	ctx.Log().Infof("AppendDexOverrides: DNSDomain returned %s", dnsSubDomain)

	host := "dex." + dnsSubDomain

	kvs = append(kvs, bom.KeyValue{
		Key:       configIssuer,
		Value:     httpPrefix + host,
		SetString: true,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:       dnsTarget,
		Value:     host,
		SetString: true,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   hostsHost,
		Value: host,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   tlsHosts,
		Value: host,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:       tlsSecret,
		Value:     dexTLSSecret,
		SetString: true,
	})

	kvs = append(kvs, bom.KeyValue{
		Key:   ingressClassKey,
		Value: vzconfig.GetIngressClassName(ctx.EffectiveCR()),
	})

	// Add override for static user
	staticUserData, err := populateStaticPasswords(ctx)
	if err != nil {
		ctx.Log().Errorf("Component Dex failed to configure static users: %v", err)
		return nil, err
	}
	userOverridePattern := tmpFilePrefix + "user-" + "*." + tmpSuffix
	userOverridesFile, err := generateOverridesFile(staticUserData.Bytes(), userOverridePattern)
	if err != nil {
		return kvs, fmt.Errorf("failed generating Dex overrides file: %v", err)
	}
	kvs = append(kvs, bom.KeyValue{Value: userOverridesFile, IsFile: true})

	staticClientData, err := populateStaticClients(ctx, dnsSubDomain)
	if err != nil {
		ctx.Log().Errorf("Component Dex failed to configure static clients: %v", err)
		return nil, err
	}
	clientOverridePattern := tmpFilePrefix + "client-" + "*." + tmpSuffix
	clientOverridesFile, err := generateOverridesFile(staticClientData.Bytes(), clientOverridePattern)
	if err != nil {
		return kvs, fmt.Errorf("failed generating Dex overrides file: %v", err)
	}

	// Append any installArgs overrides
	kvs = append(kvs, bom.KeyValue{Value: clientOverridesFile, IsFile: true})
	return kvs, nil
}

// preInstallUpgrade handles pre-install and pre-upgrade processing for the Dex Component
func preInstallUpgrade(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Dex preInstallUpgrade dry run")
		return nil
	}

	// Create the dex namespace if not already created
	ctx.Log().Debugf("Creating namespace %s for Dex", constants.DexNamespace)
	return ensureDexNamespace(ctx)
}

// ensureDexNamespace ensures that the dex namespace is created with the right labels.
func ensureDexNamespace(ctx spi.ComponentContext) error {
	// Create the dex namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.DexNamespace,
		},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), namespace, func() error {
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		namespace.Labels[v8oconst.LabelVerrazzanoNamespace] = constants.DexNamespace
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// getDNSDomain returns the DNS Domain
func getDNSDomain(c client.Client, vz *vzapi.Verrazzano) (string, error) {
	var dnsDomain string
	dnsSuffix, err := vzconfig.GetDNSSuffix(c, vz)
	if err != nil {
		return "", err
	}
	if vz != nil {
		dnsDomain = fmt.Sprintf("%s.%s", vz.Spec.EnvironmentName, dnsSuffix)
	} else {
		dnsDomain = dnsSuffix
	}
	return dnsDomain, nil
}

// updateDexIngress updates the Ingress
func updateDexIngress(ctx spi.ComponentContext) error {
	ingress := networkv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: constants.DexIngress, Namespace: constants.DexNamespace},
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &ingress, func() error {
		dnsSuffix, _ := vzconfig.GetDNSSuffix(ctx.Client(), ctx.EffectiveCR())
		ingress.Annotations["cert-manager.io/common-name"] = fmt.Sprintf("%s.%s.%s",
			ComponentName, ctx.EffectiveCR().Spec.EnvironmentName, dnsSuffix)
		ingress.Annotations["cert-manager.io/cluster-issuer"] = v8oconst.VerrazzanoClusterIssuerName
		// update target annotation on Dex Ingress for external DNS
		if vzcr.IsExternalDNSEnabled(ctx.EffectiveCR()) {
			dnsSubDomain, err := vzconfig.BuildDNSDomain(ctx.Client(), ctx.EffectiveCR())
			if err != nil {
				return err
			}
			ingressTarget := fmt.Sprintf("verrazzano-ingress.%s", dnsSubDomain)
			ctx.Log().Debugf("updateIngress: Updating updateIngress Ingress with ingressTarget = %s", ingressTarget)
			ingress.Annotations["external-dns.alpha.kubernetes.io/target"] = ingressTarget
		}
		return nil
	})
	ctx.Log().Debugf("updateIngress: Dex ingress operation result: %v", err)
	return err
}

// populateStaticPasswords populates the data for the admin user, created as static password  in Dex
func populateStaticPasswords(ctx spi.ComponentContext) (bytes.Buffer, error) {
	var b bytes.Buffer
	t, err := template.New("").Parse(staticPasswordTemplate)
	if err != nil {
		return b, fmt.Errorf("failed parsing static password template: %v", err)
	}

	err = t.Execute(&b, nil)
	if err != nil {
		return b, fmt.Errorf("failed applying static password template: %v", err)
	}

	data := userData{}
	err = populateAdminUserData(ctx, &data)
	if err != nil {
		return b, fmt.Errorf("failed populating user data for admin: %v", err)
	}

	t, err = template.New("").Parse(passwordTemplate)
	if err != nil {
		return b, fmt.Errorf("failed parsing password template: %v", err)
	}

	err = t.Execute(&b, data)
	if err != nil {
		return b, fmt.Errorf("failed applying password template: %v", err)
	}
	return b, nil
}

// populateAdminUserData populates the data for the admin user
func populateAdminUserData(ctx spi.ComponentContext, data *userData) error {
	secret := &corev1.Secret{}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: constants.VerrazzanoSystemNamespace,
		Name:      constants.Verrazzano,
	}, secret)
	if err != nil {
		ctx.Log().Errorf("Component Dex failed to get the Verrazzano password %s/%s: %v",
			constants.VerrazzanoSystemNamespace, constants.Verrazzano, err)
		return err
	}

	vzUser := secret.Data[adminUsernameKey]
	vzPwd := secret.Data[adminPasswordKey]

	// Dex expects bcrypt hash of the password
	pwdHash, err := generateBCCryptHash(ctx, vzPwd)
	if err != nil {
		return err
	}
	data.Hash = pwdHash
	data.UserName = string(vzUser)
	data.UserID = uuid.New().String()

	// Setting the verrazzano user for e-mail. There is no validation for e-mail in Dex as of now
	// This is used to prompt for the user-name in the Dex screen, even though the input is e-mail.
	data.Email = string(vzUser)
	return nil
}

// generateBCCryptHash generates the bcrypt hash of the password
func generateBCCryptHash(ctx spi.ComponentContext, password []byte) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		ctx.Log().Errorf("Component Dex failed to generate bcrypt password %v", err)
		return "", err
	}
	return string(hashedPassword), nil
}

// populateStaticClients populates the helm overrides to configure clients verrazzano-pkce and verrazzano-pg
func populateStaticClients(ctx spi.ComponentContext, dnsSubDomain string) (bytes.Buffer, error) {
	var b bytes.Buffer
	t, err := template.New("").Parse(staticClientTemplate)

	if err != nil {
		return b, fmt.Errorf("failed parsing static client template: %v", err)
	}

	err = t.Execute(&b, nil)
	if err != nil {
		return b, fmt.Errorf("failed applying static client template: %v", err)
	}

	redirectURIs, err := populateRedirectURIs(pkceClientUrisTemplate, dnsSubDomain)
	if err != nil {
		return b, fmt.Errorf("failed populating redirect URIs for client:%s :%v", pkceClient, err)
	}

	pkceSecret := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      pkceClient,
	}
	cs, err := generateClientSecret(ctx, pkceSecret)
	if err != nil {
		return b, fmt.Errorf("failed generating client secret for client:%s :%v", pkceClient, err)
	}

	cData := clientData{}
	cData.ClientID = pkceClient
	cData.ClientName = pkceClient
	cData.ClientSecret = cs
	cData.RedirectURIs = redirectURIs

	t, err = template.New("").Parse(clientTemplate)
	if err != nil {
		return b, fmt.Errorf("failed parsing static client template: %v", err)
	}

	err = t.Execute(&b, cData)
	if err != nil {
		return b, fmt.Errorf("failed applying static client template: %v", err)
	}

	pgSecret := types.NamespacedName{
		Namespace: ComponentNamespace,
		Name:      pgClient,
	}
	cs, err = generateClientSecret(ctx, pgSecret)
	if err != nil {
		return b, fmt.Errorf("failed generating client secret for client:%s :%v", pkceClient, err)
	}

	cData = clientData{}
	cData.ClientID = pgClient
	cData.ClientName = pgClient
	cData.ClientSecret = cs
	cData.RedirectURIs = ""

	t, err = template.New("").Parse(clientTemplate)
	if err != nil {
		return b, fmt.Errorf("failed parsing static client template: %v", err)
	}
	err = t.Execute(&b, cData)
	if err != nil {
		return b, fmt.Errorf("failed applying static client template: %v", err)
	}
	return b, nil
}

// populateRedirectURIs populates the redirect URIs for the given template
func populateRedirectURIs(tmpl, dnsSubDomain string) (string, error) {
	data := redirectURIsData{}
	data.DNSSubDomain = dnsSubDomain
	var b bytes.Buffer
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed parsing the template: %v", err)
	}

	err = t.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("failed applying the template: %v", err)
	}
	return b.String(), nil
}

// generateOverridesFile creates the helm overrides file for Dex, using the contents
func generateOverridesFile(contents []byte, filePattern string) (string, error) {
	file, err := os.CreateTemp(os.TempDir(), filePattern)
	if err != nil {
		return "", err
	}

	overridesFileName := file.Name()
	if err := writeFileFunc(overridesFileName, contents, fs.ModeAppend); err != nil {
		return "", err
	}
	return overridesFileName, nil
}

// generateClientSecret creates the secret for the given client
func generateClientSecret(ctx spi.ComponentContext, clientName types.NamespacedName) (string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: clientName.Name, Namespace: clientName.Namespace},
	}
	err := ctx.Client().Get(context.TODO(), client.ObjectKey{
		Namespace: clientName.Namespace,
		Name:      clientName.Name,
	}, secret)

	// If the secret doesn't exist, create it
	if err != nil {
		pw, err := vzpassword.GeneratePassword(12)
		if err != nil {
			return "", fmt.Errorf("failed to generate a password for the client %s: %v", clientName.Name, err)
		}
		_, err = controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
			secret.Data = map[string][]byte{
				dexClientSecret: []byte(pw),
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("unable to create or update the secret for the client %s: %v", clientName.Name, err)
		}
		ctx.Log().Infof("Created secret %s successfully", clientName)
		return pw, nil
	}
	return "", err
}
