// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package gateway

import (
	"fmt"
	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	consts "github.com/verrazzano/verrazzano/tools/oam-converter/pkg/constants"
	istio "istio.io/api/networking/v1beta1"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
)

// createGatewaySecret will create a certificate that will be embedded in an secret or leverage an existing secret
// if one is configured in the ingress.
func CreateGatewaySecret(trait *vzapi.IngressTrait, hostsForTrait []string) string {
	var secretName string

	if trait.Spec.TLS != (vzapi.IngressSecurity{}) {
		secretName = validateConfiguredSecret(trait)
	} else {
		buildLegacyCertificateName(trait)
		buildLegacyCertificateSecretName(trait)
		secretName = createGatewayCertificate(trait, hostsForTrait)
	}
	return secretName
}

// The function createGatewayCertificate generates a certificate that the cert manager can use to generate a certificate
// that is embedded in a secret. The gateway will use the secret to offer TLS/HTTPS endpoints for installed applications.
// Each application will generate a single gateway. The application-wide gateway will be used to route the produced
// virtual services
func createGatewayCertificate(trait *vzapi.IngressTrait, hostsForTrait []string) string {
	//ensure trait does not specify hosts.  should be moved to ingress trait validating webhook
	for _, rule := range trait.Spec.Rules {
		if len(rule.Hosts) != 0 {
			print("Host(s) specified in the trait rules will likely not correlate to the generated certificate CN." +
				"Please redeploy after removing the hosts or specifying a secret with the given hosts in its SAN list")
			break
		}
	}
	certName := buildCertificateName(trait)
	secretName := buildCertificateSecretName(trait)
	certificate := &certapiv1.Certificate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Certificate",
			APIVersion: consts.CertificateAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      certName,
		}}
	certificate.Spec = certapiv1.CertificateSpec{
		DNSNames:   hostsForTrait,
		SecretName: secretName,
		IssuerRef: certv1.ObjectReference{
			Name: consts.VerrazzanoClusterIssuer,
			Kind: "ClusterIssuer",
		},
	}

	return secretName
}

// buildCertificateSecretName will construct a cert secret name from the trait.
func buildCertificateSecretName(trait *vzapi.IngressTrait) string {
	return fmt.Sprintf("%s-%s-cert-secret", trait.Namespace, trait.Name)
}

// buildCertificateName will construct a cert name from the trait.
func buildCertificateName(trait *vzapi.IngressTrait) string {
	return fmt.Sprintf("%s-%s-cert", trait.Namespace, trait.Name)
}

// buildLegacyCertificateName will generate a cert name
func buildLegacyCertificateName(trait *vzapi.IngressTrait) string {
	appName, ok := trait.Labels[oam.LabelAppName]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s-%s-cert", trait.Namespace, appName)
}

// buildLegacyCertificateSecretName will generate a cert secret name
func buildLegacyCertificateSecretName(trait *vzapi.IngressTrait) string {
	appName, ok := trait.Labels[oam.LabelAppName]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s-%s-cert-secret", trait.Namespace, appName)
}

// validateConfiguredSecret ensures that a secret is specified and the trait rules specify a "hosts" setting.  The
// specification of a secret implies that a certificate was created for specific hosts that differ than the host names
// generated by the runtime (when no hosts are specified).
func validateConfiguredSecret(trait *vzapi.IngressTrait) string {
	secretName := trait.Spec.TLS.SecretName
	return secretName
}

// buildGatewayName will generate a gateway name from the namespace and application name of the provided trait. Returns
// an error if the app name is not available.
func BuildGatewayName(trait *vzapi.IngressTrait) (string, error) {

	gwName := fmt.Sprintf("%s-%s-gw", trait.Namespace, trait.Name)
	return gwName, nil
}

// mutateGateway mutates the output Gateway child resource.
func mutateGateway(traitName string, gateway *vsapi.Gateway, trait *vzapi.IngressTrait, hostsForTrait []string, secretName string) error {

	server := &istio.Server{
		Name:  trait.Name,
		Hosts: hostsForTrait,
		Port: &istio.Port{
			Name:     formatGatewayServerPortName(traitName),
			Number:   443,
			Protocol: consts.HTTPSProtocol,
		},
		Tls: &istio.ServerTLSSettings{
			Mode:           istio.ServerTLSSettings_SIMPLE,
			CredentialName: secretName,
		},
	}
	gateway.Spec.Servers = updateGatewayServersList(gateway.Spec.Servers, server)

	// Set the spec content.
	gateway.Spec.Selector = map[string]string{"istio": "ingressgateway"}

	fmt.Println("gateway", gateway)
	directoryPath := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/"
	fileName := "gw.yaml"
	filePath := filepath.Join(directoryPath, fileName)

	gatewayYaml, err := yaml.Marshal(gateway)
	if err != nil {
		fmt.Printf("Failed to marshal: %v\n", err)
		return err
	}
	// Write the YAML content to the file
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open file: %v\n", err)
		return err
	}
	defer file.Close()

	// Append the YAML content to the file
	_, err = file.Write(gatewayYaml)
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	_, err = file.WriteString("---\n")
	if err != nil {
		fmt.Printf("Failed to write to file: %v\n", err)
		return err
	}
	return nil
}

func formatGatewayServerPortName(traitName string) string {
	return fmt.Sprintf("https-%s", traitName)
}

// updateGatewayServersList Update/add the Server entry for the IngressTrait to the gateway servers list
// Each gateway server entry has a TLS field for the certificate.  This corresponds to the IngressTrait TLS field.
func updateGatewayServersList(servers []*istio.Server, server *istio.Server) []*istio.Server {
	if len(servers) == 0 {
		servers = append(servers, server)
		return servers
	}
	if len(servers) == 1 && len(servers[0].Name) == 0 && servers[0].Port.Name == "https" {

		// - replace the empty name server with the named one
		servers[0] = server

		return servers
	}
	for index, existingServer := range servers {
		if existingServer.Name == server.Name {

			servers[index] = server
			return servers
		}
	}
	servers = append(servers, server)
	return servers
}

// createGateway creates the Gateway child resource of the trait.
func CreateGateway(traitName string, trait *vzapi.IngressTrait, hostsForTrait []string, gwName string, secretName string) (*vsapi.Gateway, error) {
	// Create a gateway populating only gwName metadata.
	// This is used as default if the gateway needs to be created.
	gateway := &vsapi.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: consts.GatewayAPIVersion,
			Kind:       "Gateway"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: trait.Namespace,
			Name:      gwName}}

	err := mutateGateway(traitName, gateway, trait, hostsForTrait, secretName)
	if err != nil {
		return nil, err
	}

	return gateway, nil
}
