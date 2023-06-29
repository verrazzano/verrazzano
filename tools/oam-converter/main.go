// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	certapiv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certv1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"google.golang.org/protobuf/types/known/durationpb"
	"io/ioutil"
	istionet "istio.io/api/networking/v1alpha3"
	istio "istio.io/api/networking/v1beta1"
	"istio.io/api/security/v1beta1"
	v1beta12 "istio.io/api/type/v1beta1"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	vsapi "istio.io/client-go/pkg/apis/networking/v1beta1"
	clisecurity "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
	"strings"
	"time"
)

var (
	weblogicPortNames = []string{"tcp-cbt", "tcp-ldap", "tcp-iiop", "tcp-snmp", "tcp-default", "tls-ldaps",
		"tls-default", "tls-cbts", "tls-iiops", "tcp-internal-t3", "internal-t3"}
)

// var cli client.Reader
func main() {
	//Read OAM File
	yamlData, err := ioutil.ReadFile("examples/hello-helidon/hello-helidon-app.yaml")
	if err != nil {
		fmt.Println("Failed to read YAML file:", err)
		return
	}

	// Create a map to store the parsed OAM YAML input data
	yamlMap := make(map[string]interface{})

	// Unmarshal the OAM YAML input data into the map
	err = yaml.Unmarshal(yamlData, &yamlMap)
	if err != nil {
		log.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	// Access and work with the YAML data
	ingressTraits := []*vzapi.IngressTrait{} //Array of ingresstraits
	metricsTraits := []*vzapi.MetricsTrait{} //Array of metricstraits

	// Access nested objects within the YAML data and extract traits by checking the kind of the object
	components := yamlMap["spec"].(map[string]interface{})["components"].([]interface{})
	for _, component := range components {
		componentMap := component.(map[string]interface{})
		componentTraits, ok := componentMap["traits"].([]interface{})
		if ok {
			for _, trait := range componentTraits {
				traitMap := trait.(map[string]interface{})
				traitSpec := traitMap["trait"].(map[string]interface{})
				traitKind := traitSpec["kind"].(string)
				if traitKind == "IngressTrait" {
					ingressTrait := &vzapi.IngressTrait{}
					traitJSON, err := json.Marshal(traitSpec)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(traitJSON, ingressTrait)
					ingressTrait.Name = yamlMap["metadata"].(map[string]interface{})["name"].(string)
					ingressTrait.Namespace = yamlMap["metadata"].(map[string]interface{})["namespace"].(string)
					if err != nil {
						log.Fatalf("Failed to unmarshal trait: %v", err)
					}

					ingressTraits = append(ingressTraits, ingressTrait)
				}
				if traitKind == "MetricsTrait" {
					metricsTrait := &vzapi.MetricsTrait{}
					traitJSON, err := json.Marshal(traitSpec)

					if err != nil {
						log.Fatalf("Failed to marshal trait: %v", err)
					}

					err = json.Unmarshal(traitJSON, metricsTrait)
					metricsTrait.Name = yamlMap["metadata"].(map[string]interface{})["name"].(string)
					metricsTrait.Namespace = yamlMap["metadata"].(map[string]interface{})["namespace"].(string)
					if err != nil {
						log.Fatalf("Failed to unmarshal trait: %v", err)
					}
					metricsTraits = append(metricsTraits, metricsTrait)
				}

			}
		}
	}

	//Create child resources of each metric trait
	for _, trait := range metricsTraits {

		fmt.Printf("Trait API Version: %s\n", trait.APIVersion)
		fmt.Printf("Trait name: %s\n", trait.Name)
		createMetricsChildResources(trait)
		//Put metricsTrait method
	}

	//Create child resources of each ingress trait
	for _, trait := range ingressTraits {

		fmt.Printf("Trait API Version: %s\n", trait.APIVersion)
		fmt.Printf("Trait name: %s\n", trait.Name)
		createIngressChildResources(trait)
	}

}
func createMetricsChildResources(metricstrait *vzapi.MetricsTrait) {

	createService()
	createServiceMonitor(metricstrait)
}
// createChildResources creates the Gateway, VirtualService, DestinationRule and AuthorizationPolicy resources
func createIngressChildResources(ingresstrait *vzapi.IngressTrait) {
	rules := ingresstrait.Spec.Rules
	// If there are no rules, create a single default rule
	if len(rules) == 0 {
		rules = []vzapi.IngressRule{{}}
	}

	// Create a list of unique hostnames across all rules in the trait
	allHostsForTrait := coallateAllHostsForTrait(ingresstrait)

	// Generate the certificate and secret for all hosts in the trait rules
	secretName := createGatewaySecret(ingresstrait, allHostsForTrait)
	if secretName != "" {
		gwName, err := buildGatewayName(ingresstrait)
		if err != nil {
			print(err)
		} else {
			//creating gateway
			gateway, err := createGateway(ingresstrait, allHostsForTrait, gwName, secretName)
			if err != nil {
				print(err)
			}
			for index, rule := range rules {
				// Find the services associated with the trait in the application configuration.
				var services []*corev1.Service

				//services, err := fetchServicesFromTrait(ingresstrait)
				//if err != nil {
				//	print(err)
				//} else if len(services) == 0 {
				//	// This will be the case if the service has not started yet so we requeue and try again.
				//	print(err)
				//}

				vsHosts, err := createHostsFromIngressTraitRule(rule, ingresstrait)
				if err != nil {
					print(err)
				}
				vsName := fmt.Sprintf("%s-rule-%d-vs", ingresstrait.Name, index)
				drName := fmt.Sprintf("%s-rule-%d-dr", ingresstrait.Name, index)
				authzPolicyName := fmt.Sprintf("%s-rule-%d-authz", ingresstrait.Name, index)
				createVirtualService(ingresstrait, rule, vsHosts, vsName, services, gateway)
				createDestinationRule(ingresstrait, rule, drName, services)
				createAuthorizationPolicies(ingresstrait, rule, authzPolicyName, allHostsForTrait)
			}
		}
	}
}
//creates Server Monitor Instance
func createServiceMonitor(trait *vzapi.MetricsTrait){
	// Creating a service monitor with name and namespace
	pmName, err := createServiceMonitorName(trait, 0)
	if err != nil {
		print(err)
	}

	// Fetch the secret by name if it is provided in either the trait or the trait defaults.
	//do I need fetchSourceCrednentials and Istio enabled
	var workload *unstructured.Unstructured
	secret, err := fetchSourceCredentialsSecretIfRequired(trait, traitDefaults, workload)
	if err != nil {
		print(err)
	}

	//find workload
	//wlsWorkload, err := isWLSWorkload(workload)
	//if err != nil {
	//	print(err)
	//}

	//vzPromLabels := !wlsWorkload

	//scrapeInfo := metrics.ScrapeInfo{
	//	Ports:              len(getPortSpecs(trait, traitDefaults)),
	//	BasicAuthSecret:    secret,
	//	IstioEnabled:       &useHTTPS,
	//	VZPrometheusLabels: &vzPromLabels,
	//	ClusterName:        clusters.GetClusterName(ctx, r.Client),
	//}
	//
	//// Fill in the scrape info if it is populated in the trait
	//if trait.Spec.Path != nil {
	//	scrapeInfo.Path = trait.Spec.Path
	//}
	//
	//// Populate the keep labels to match the oam pod labels
	//scrapeInfo.KeepLabels = map[string]string{
	//	"__meta_kubernetes_pod_label_app_oam_dev_name":      trait.Labels[oam.LabelAppName],
	//	"__meta_kubernetes_pod_label_app_oam_dev_component": trait.Labels[oam.LabelAppComponent],
	//}
	//
	//serviceMonitor := promoperapi.ServiceMonitor{}
	//serviceMonitor.SetName(pmName)
	//serviceMonitor.SetNamespace(workload.GetNamespace())
	//result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &serviceMonitor, func() error {
	//	return metrics.PopulateServiceMonitor(scrapeInfo, &serviceMonitor, log)
	//})

}

func createService(){

}
func createServiceMonitorName(trait *vzapi.MetricsTrait, portNum int) (string, error) {
	sname, err := createJobOrServiceMonitorName(trait, portNum)
	if err != nil {
		return "", err
	}
	return strings.Replace(sname, "_", "-", -1), nil
}
func createJobOrServiceMonitorName(trait *vzapi.MetricsTrait, portNum int) (string, error) {
	namespace := getNamespaceFromObjectMetaOrDefault(trait.ObjectMeta)
	app, found := trait.Labels[oam.LabelAppName]
	if !found {
		return "", fmt.Errorf("metrics trait missing application name label")
	}
	comp, found := trait.Labels[oam.LabelAppComponent]
	if !found {
		return "", fmt.Errorf("metrics trait missing component name label")
	}
	portStr := ""
	if portNum > 0 {
		portStr = fmt.Sprintf("_%d", portNum)
	}

	finalName := fmt.Sprintf("%s_%s_%s%s", app, namespace, comp, portStr)
	// Check for Kubernetes name length requirement
	if len(finalName) > 63 {
		finalName = fmt.Sprintf("%s_%s%s", app, namespace, portStr)
		if len(finalName) > 63 {
			return finalName[:63], nil
		}
	}
	return finalName, nil
}
func getNamespaceFromObjectMetaOrDefault(meta metav1.ObjectMeta) string {
	name := meta.Namespace
	if name == "" {
		return "default"
	}
	return name
}
//if kind == app config && doesnt specify, create metricstrait
func fetchSourceCredentialsSecretIfRequired(trait *vzapi.MetricsTrait, traitDefaults *vzapi.MetricsTraitSpec, workload *unstructured.Unstructured){
	secretName := trait.Spec.Secret
	// If no secret name explicitly provided use the default secret name.
	if secretName == nil && traitDefaults != nil {
		secretName = traitDefaults.Secret
	}
	// If neither an explicit or default secret name provided do not fetch a secret.
	if secretName == nil {
		return nil, nil
	}
	// Use the workload namespace for the secret to fetch.
	secretNamespace, found, err := unstructured.NestedString(workload.Object, "metadata", "namespace")
	if err != nil {
		return nil, fmt.Errorf("failed to determine namespace for secret %s: %w", *secretName, err)
	}
	if !found {
		return nil, fmt.Errorf("failed to find namespace for secret %s", *secretName)
	}
	// Fetch the secret.
	secretKey := client.ObjectKey{Namespace: secretNamespace, Name: *secretName}
	secretObj := k8score.Secret{}
	err = cli.Get(ctx, secretKey, &secretObj)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret %v: %w", secretKey, err)
	}
	return &secretObj, nil
}
func useHTTPSForScrapeTarget(trait *vzapi.MetricsTrait) (bool, error) {
	if trait.Spec.WorkloadReference.Kind == "VerrazzanoCoherenceWorkload" || trait.Spec.WorkloadReference.Kind == "Coherence" {
		return false, nil
	}
	// Get the namespace resource that the MetricsTrait is deployed to
	namespace := &k8score.Namespace{}

	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		return true, nil
	}
	return false, nil
}
// creates Authorization Policy
func createAuthorizationPolicies(trait *vzapi.IngressTrait, rule vzapi.IngressRule, namePrefix string, hosts []string) {
	// If any path needs an AuthorizationPolicy then add one for every path
	var addAuthPolicy bool
	for _, path := range rule.Paths {
		if path.Policy != nil {
			addAuthPolicy = true
		}
	}
	for _, path := range rule.Paths {
		if addAuthPolicy {
			requireFrom := true

			// Add a policy rule if one is missing
			if path.Policy == nil {
				path.Policy = &vzapi.AuthorizationPolicy{
					Rules: []*vzapi.AuthorizationRule{{}},
				}
				// No from field required, this is just a path being added
				requireFrom = false
			}

			pathSuffix := strings.Replace(path.Path, "/", "", -1)
			policyName := namePrefix
			if pathSuffix != "" {
				policyName = fmt.Sprintf("%s-%s", policyName, pathSuffix)
			}

			authzPolicy := &clisecurity.AuthorizationPolicy{
				TypeMeta: metav1.TypeMeta{
					Kind:       "AuthorizationPolicy",
					APIVersion: "security.istio.io/v1beta1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      policyName,
					Namespace: constants.IstioSystemNamespace,
					Labels:    map[string]string{constants.LabelIngressTraitNsn: getIngressTraitNsn(trait.Namespace, trait.Name)},
				},
			}
			mutateAuthorizationPolicy(authzPolicy, path.Policy, path.Path, hosts, requireFrom)
		}
	}
}

func getIngressTraitNsn(namespace string, name string) string {
	return fmt.Sprintf("%s-%s", namespace, name)
}

// mutateAuthorizationPolicy changes the destination rule based upon a trait's configuration
func mutateAuthorizationPolicy(authzPolicy *clisecurity.AuthorizationPolicy, vzPolicy *vzapi.AuthorizationPolicy, path string, hosts []string, requireFrom bool) {
	policyRules := make([]*v1beta1.Rule, len(vzPolicy.Rules))
	var err error
	for i, authzRule := range vzPolicy.Rules {
		policyRules[i], err = createAuthorizationPolicyRule(authzRule, path, hosts, requireFrom)
		if err != nil {
			print(err)
		}
	}

	authzPolicy.Spec = v1beta1.AuthorizationPolicy{
		Selector: &v1beta12.WorkloadSelector{
			MatchLabels: map[string]string{"istio": "ingressgateway"},
		},
		Rules: policyRules,
	}
	fmt.Println("AuthorizationPolicy", authzPolicy)
}

// createAuthorizationPolicyRule uses the provided information to create an istio authorization policy rule
func createAuthorizationPolicyRule(rule *vzapi.AuthorizationRule, path string, hosts []string, requireFrom bool) (*v1beta1.Rule, error) {
	authzRule := v1beta1.Rule{}

	if requireFrom && rule.From == nil {
		return nil, fmt.Errorf("Authorization Policy requires 'From' clause")
	}
	if rule.From != nil {
		authzRule.From = []*v1beta1.Rule_From{
			{Source: &v1beta1.Source{
				RequestPrincipals: rule.From.RequestPrincipals},
			},
		}
	}

	if len(path) > 0 {
		authzRule.To = []*v1beta1.Rule_To{{
			Operation: &v1beta1.Operation{
				Hosts: hosts,
				Paths: []string{path},
			},
		}}
	}

	if rule.When != nil {
		conditions := []*v1beta1.Condition{}
		for _, vzCondition := range rule.When {
			condition := &v1beta1.Condition{
				Key:    vzCondition.Key,
				Values: vzCondition.Values,
			}
			conditions = append(conditions, condition)
		}
		authzRule.When = conditions
	}

	return &authzRule, nil
}

// createOfUpdateDestinationRule creates or updates the DestinationRule.
func createDestinationRule(trait *vzapi.IngressTrait, rule vzapi.IngressRule, name string, services []*corev1.Service) {
	if rule.Destination.HTTPCookie != nil {
		destinationRule := &istioclient.DestinationRule{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "networking.istio.io/v1alpha3",
				Kind:       "DestinationRule"},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: trait.Namespace,
				Name:      name},
		}
		namespace := &corev1.Namespace{}
		//namespaceErr := cli.Get(client.ObjectKey{Namespace: "", Name: trait.Namespace}, namespace)
		//if namespaceErr != nil {
		//
		//}
		fmt.Println("destinationRule", destinationRule)
		mutateDestinationRule(destinationRule, rule, services, namespace)

	}
}

// mutateDestinationRule changes the destination rule based upon a traits configuration
func mutateDestinationRule(destinationRule *istioclient.DestinationRule, rule vzapi.IngressRule, services []*corev1.Service, namespace *corev1.Namespace) {
	dest, err := createDestinationFromRuleOrService(rule, services)

	if err != nil {
		print(err)
	}

	mode := istionet.ClientTLSSettings_DISABLE
	value, ok := namespace.Labels["istio-injection"]
	if ok && value == "enabled" {
		mode = istionet.ClientTLSSettings_ISTIO_MUTUAL
	}
	destinationRule.Spec = istionet.DestinationRule{
		Host: dest.Destination.Host,
		TrafficPolicy: &istionet.TrafficPolicy{
			Tls: &istionet.ClientTLSSettings{
				Mode: mode,
			},
			LoadBalancer: &istionet.LoadBalancerSettings{
				LbPolicy: &istionet.LoadBalancerSettings_ConsistentHash{
					ConsistentHash: &istionet.LoadBalancerSettings_ConsistentHashLB{
						HashKey: &istionet.LoadBalancerSettings_ConsistentHashLB_HttpCookie{
							HttpCookie: &istionet.LoadBalancerSettings_ConsistentHashLB_HTTPCookie{
								Name: rule.Destination.HTTPCookie.Name,
								Path: rule.Destination.HTTPCookie.Path,
								Ttl:  durationpb.New(rule.Destination.HTTPCookie.TTL * time.Second)},
						},
					},
				},
			},
		},
	}
	fmt.Println("DestinationRule", destinationRule)
}

// createGateway creates the Gateway child resource of the trait.
func createGateway(trait *vzapi.IngressTrait, hostsForTrait []string, gwName string, secretName string) (*vsapi.Gateway, error) {
	// Create a gateway populating only gwName metadata.
	// This is used as default if the gateway needs to be created.
	gateway := &vsapi.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1alpha3",
			Kind:       "Gateway"},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: trait.Namespace,
			Name:      gwName}}

	err := mutateGateway(gateway, trait, hostsForTrait, secretName)
	if err != nil {
		return nil, err
	}

	return gateway, nil
}

// buildGatewayName will generate a gateway name from the namespace and application name of the provided trait. Returns
// an error if the app name is not available.
func buildGatewayName(trait *vzapi.IngressTrait) (string, error) {

	gwName := fmt.Sprintf("%s-%s-gw", trait.Namespace, trait.Name)
	return gwName, nil
}

// mutateGateway mutates the output Gateway child resource.
func mutateGateway(gateway *vsapi.Gateway, trait *vzapi.IngressTrait, hostsForTrait []string, secretName string) error {

	server := &istio.Server{
		Name:  trait.Name,
		Hosts: hostsForTrait,
		Port: &istio.Port{
			Name:     formatGatewaySeverPortName(trait.Name),
			Number:   443,
			Protocol: "HTTPS",
		},
		Tls: &istio.ServerTLSSettings{
			Mode:           istio.ServerTLSSettings_SIMPLE,
			CredentialName: secretName,
		},
	}
	gateway.Spec.Servers = updateGatewayServersList(gateway.Spec.Servers, server)

	// Set the spec content.
	gateway.Spec.Selector = map[string]string{"istio": "ingressgateway"}

	//gatewayYaml, err := yaml.Marshal(&gateway)
	//if err != nil {
	//	fmt.Printf("Error while Marshaling. %v", err)
	//}
	//
	//fileName := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/test.yaml"
	//file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//if err != nil {
	//	fmt.Printf("Failed to open YAML file: %v\n", err)
	//	return err
	//}
	//
	//// Write the YAML content at the end of the file
	//_, err = file.Write(gatewayYaml)
	//if err != nil {
	//	fmt.Printf("Failed to write YAML content: %v\n", err)
	//	return nil
	//}
	//yamlContent := "---\n"
	//
	//// Write the YAML content to a file
	//_, err = file.Write([]byte(yamlContent))

	fmt.Println("gateway", gateway)
	return nil
}

func formatGatewaySeverPortName(traitName string) string {
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

// createGatewaySecret will create a certificate that will be embedded in an secret or leverage an existing secret
// if one is configured in the ingress.
func createGatewaySecret(trait *vzapi.IngressTrait, hostsForTrait []string) string {
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
			APIVersion: "cert-manager.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "istio-system",
			Name:      certName,
		}}
	certificate.Spec = certapiv1.CertificateSpec{
		DNSNames:   hostsForTrait,
		SecretName: secretName,
		IssuerRef: certv1.ObjectReference{
			Name: "verrazzano-cluster-issuer",
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
	//if secretName != "" {
	//	// if a secret is specified then host(s) must be provided for all rules
	//	for _, rule := range trait.Spec.Rules {
	//		if len(rule.Hosts) == 0 {
	//			//err := errors.New("all rules must specify at least one host when a secret is specified for TLS transport")
	//			ref := vzapi.QualifiedResourceRelation{APIVersion: "v1", Kind: "Secret", Name: secretName, Role: "secret"}
	//			status.Relations = append(status.Relations, ref)
	//			status.Errors = append(status.Errors, err)
	//			status.Results = append(status.Results, controllerutil.OperationResultNone)
	//			return ""
	//		}
	//	}
	//}
	return secretName
}

func coallateAllHostsForTrait(trait *vzapi.IngressTrait) []string {
	allHosts := []string{}
	var err error
	for _, rule := range trait.Spec.Rules {
		if allHosts, err = createHostsFromIngressTraitRule(rule, trait, allHosts...); err != nil {
			print(err)
		}
	}
	return allHosts
}

// createHostsFromIngressTraitRule creates an array of hosts from an ingress rule, appending to an optionally provided input list
// - It filters out wildcard hosts or hosts that are empty.
// - If there are no valid hosts provided for the rule, then a DNS host name is automatically generated and used.
// - A hostname can only appear once
func createHostsFromIngressTraitRule(rule vzapi.IngressRule, trait *vzapi.IngressTrait, toList ...string) ([]string, error) {
	validHosts := toList
	useDefaultHost := true
	for _, h := range rule.Hosts {
		h = strings.TrimSpace(h)
		if _, hostAlreadyPresent := findHost(validHosts, h); hostAlreadyPresent {
			// Avoid duplicates
			useDefaultHost = false
			continue
		}
		// Ignore empty or wildcard hostname
		if len(h) == 0 || strings.Contains(h, "*") {
			continue
		}
		h = strings.ToLower(strings.TrimSpace(h))
		validHosts = append(validHosts, h)
		useDefaultHost = false
	}
	// Add done if a host was added to the host list
	if !useDefaultHost {
		return validHosts, nil
	}

	// Generate a default hostname
	hostName, err := buildAppFullyQualifiedHostName(trait)
	if err != nil {
		return nil, err
	}
	// Only add the generated hostname if it doesn't exist in hte list
	if _, hostAlreadyPresent := findHost(validHosts, hostName); !hostAlreadyPresent {
		validHosts = append(validHosts, hostName)
	}
	return validHosts, nil
}

// buildAppFullyQualifiedHostName generates a DNS host name for the application using the following structure:
// <app>.<namespace>.<dns-subdomain>  where
//
//	app is the OAM application name
//	namespace is the namespace of the OAM application
//	dns-subdomain is The DNS subdomain name
func buildAppFullyQualifiedHostName(trait *vzapi.IngressTrait) (string, error) {

	domainName, err := buildNamespacedDomainName(trait)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s.%s", trait.Name, domainName), nil
}

// buildNamespacedDomainName generates a domain name for the application using the following structure:
// <namespace>.<dns-subdomain>  where
//
//	namespace is the namespace of the OAM application
//	dns-subdomain is The DNS subdomain name
func buildNamespacedDomainName(trait *vzapi.IngressTrait) (string, error) {

	const externalDNSKey = "external-dns.alpha.kubernetes.io/target"
	const wildcardDomainKey = "verrazzano.io/dns.wildcard.domain"
	cfg, _ := config.GetConfig()
	cli, _ := client.New(cfg, client.Options{})

	// Extract the domain name from the Verrazzano ingress
	ingress := k8net.Ingress{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: "verrazzano-ingress", Namespace: "verrazzano-system"}, &ingress)
	if err != nil {
		return "", err
	}
	externalDNSAnno, ok := ingress.Annotations[externalDNSKey]
	if !ok || len(externalDNSAnno) == 0 {
		return "", fmt.Errorf("Annotation %s missing from Verrazzano ingress, unable to generate DNS name", externalDNSKey)
	}

	domain := externalDNSAnno[len("verrazzano-ingress")+1:]

	// Get the DNS wildcard domain from the annotation if it exist.  This annotation is only available
	// when the install is using DNS type wildcard (nip.io, sslip.io, etc.)
	suffix := ""
	wildcardDomainAnno, ok := ingress.Annotations[wildcardDomainKey]
	if ok {
		suffix = wildcardDomainAnno
	}

	// Build the domain name using Istio info
	if len(suffix) != 0 {
		domain, err = buildDomainNameForWildcard(cli, suffix)
		if err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%s.%s", trait.Namespace, domain), nil
}

// findHost searches for a host in the provided list. If found it will
// return it's key, otherwise it will return -1 and a bool of false.
func findHost(hosts []string, newHost string) (int, bool) {
	for i, host := range hosts {
		if strings.EqualFold(host, newHost) {
			return i, true
		}
	}
	return -1, false
}

// creates the VirtualService child resource of the trait.
func createVirtualService(ingresstrait *vzapi.IngressTrait, rule vzapi.IngressRule,
	allHostsForTrait []string, name string, services []*corev1.Service, gateway *vsapi.Gateway) {
	virtualService := &vsapi.VirtualService{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1alpha3",
			Kind:       "VirtualService",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ingresstrait.Namespace,
			Name:      name,
		},
	}
	mutateVirtualService(virtualService, rule, allHostsForTrait, services, gateway)
}

// mutateVirtualService mutates the output virtual service resource
func mutateVirtualService(virtualService *vsapi.VirtualService, rule vzapi.IngressRule, allHostsForTrait []string, services []*corev1.Service, gateway *vsapi.Gateway) {
	virtualService.Spec.Gateways = []string{gateway.Name}
	virtualService.Spec.Hosts = allHostsForTrait
	matches := []*istio.HTTPMatchRequest{}
	paths := getPathsFromRule(rule)
	for _, path := range paths {
		matches = append(matches, &istio.HTTPMatchRequest{
			Uri: createVirtualServiceMatchURIFromIngressTraitPath(path)})
	}
	dest, err := createDestinationFromRuleOrService(rule, services)
	if err != nil {
		print(err)
	}
	route := istio.HTTPRoute{
		Match: matches,
		Route: []*istio.HTTPRouteDestination{dest}}
	virtualService.Spec.Http = []*istio.HTTPRoute{&route}

	//virtualYaml, err := yaml.Marshal(&virtualService)
	//if err != nil {
	//	fmt.Printf("Error while Marshaling. %v", err)
	//}
	//
	//fileName := "/Users/vrushah/GolandProjects/verrazzano/tools/oam-converter/test.yaml"
	//file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//if err != nil {
	//	fmt.Printf("Failed to open YAML file: %v\n", err)
	//	return
	//}
	//
	//// Write the YAML content at the end of the file
	//_, err = file.Write(virtualYaml)
	//if err != nil {
	//	fmt.Printf("Failed to write YAML content: %v\n", err)
	//	return
	//}
	//yamlContent := "---\n"
	//
	//_, err = file.Write([]byte(yamlContent))

	fmt.Println("virtual-service", virtualService)
}

// createDestinationFromRuleOrService creates a destination from either the rule or the service.
// If the rule contains destination information that is used.
func createDestinationFromRuleOrService(rule vzapi.IngressRule, services []*corev1.Service) (*istio.HTTPRouteDestination, error) {
	if len(rule.Destination.Host) > 0 {
		dest := &istio.HTTPRouteDestination{Destination: &istio.Destination{Host: rule.Destination.Host}}
		if rule.Destination.Port != 0 {
			dest.Destination.Port = &istio.PortSelector{Number: rule.Destination.Port}
		}
		return dest, nil
	}

	//if rule.Destination.Port != 0 {
	return createDestinationMatchRulePort(services, rule.Destination.Port)
	//}
	//return createDestinationFromService(services) //if port not given
}

// createDestinationMatchRulePort fetches a Service matching the specified rule port and creates virtual service destination.
func createDestinationMatchRulePort(services []*corev1.Service, rulePort uint32) (*istio.HTTPRouteDestination, error) {
	var eligibleServices []*corev1.Service
	for _, service := range services {
		for _, servicePort := range service.Spec.Ports {
			if servicePort.Port == int32(rulePort) {
				eligibleServices = append(eligibleServices, service)
			}
		}
	}
	selectedService, err := selectServiceForDestination(eligibleServices)
	if err != nil {
		return nil, err
	}
	if selectedService != nil {
		dest := istio.HTTPRouteDestination{
			Destination: &istio.Destination{Host: selectedService.Name}}
		// Set the port to rule destination port
		dest.Destination.Port = &istio.PortSelector{Number: rulePort}
		return &dest, nil
	}
	return nil, fmt.Errorf("unable to select service for specified destination port %d", rulePort)
}

// selectServiceForDestination selects a Service to be used for virtual service destination.
func selectServiceForDestination(services []*corev1.Service) (*corev1.Service, error) {
	var clusterIPServices []*corev1.Service
	var allowedServices []*corev1.Service
	var allowedClusterIPServices []*corev1.Service

	// If there is only one service, return that service
	if len(services) == 1 {
		return services[0], nil
	}
	// Multiple services case
	for _, service := range services {
		if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" {
			clusterIPServices = append(clusterIPServices, service)
		}
		allowedPorts := append(getHTTPPorts(service), getWebLogicPorts(service)...)
		if len(allowedPorts) > 0 {
			allowedServices = append(allowedServices, service)
		}
		if service.Spec.ClusterIP != "" && service.Spec.ClusterIP != "None" && len(allowedPorts) > 0 {
			allowedClusterIPServices = append(allowedClusterIPServices, service)
		}
	}
	// If there is no service with cluster-IP or no service with allowed port, return an error.
	if len(clusterIPServices) == 0 && len(allowedServices) == 0 {
		return nil, fmt.Errorf("unable to select default service for destination")
	} else if len(clusterIPServices) == 1 {
		// If there is only one service with cluster IP, return that service.
		return clusterIPServices[0], nil
	} else if len(allowedClusterIPServices) == 1 {
		// If there is only one http/WebLogic service with cluster IP, return that service.
		return allowedClusterIPServices[0], nil
	} else if len(allowedServices) == 1 {
		// If there is only one http/WebLogic service, return that service.
		return allowedServices[0], nil
	}
	// In all other cases, return error.
	return nil, fmt.Errorf("unable to select the service for destination. The service port " +
		"should be named with prefix \"http\" if there are multiple services OR the IngressTrait must specify the port")
}

// getHTTPPorts returns all the service ports having the prefix "http" in their names.
func getHTTPPorts(service *corev1.Service) []corev1.ServicePort {
	var httpPorts []corev1.ServicePort
	for _, servicePort := range service.Spec.Ports {
		// Check if service port name has the http prefix
		if strings.HasPrefix(servicePort.Name, "http") {
			httpPorts = append(httpPorts, servicePort)
		}
	}
	return httpPorts
}

// getWebLogicPorts returns WebLogic ports if any present for the service. A port is evaluated as a WebLogic port if
// the port name is from the known WebLogic non-http prefixed port names used by the WebLogic operator.
func getWebLogicPorts(service *corev1.Service) []corev1.ServicePort {
	var webLogicPorts []corev1.ServicePort
	selectorMap := service.Spec.Selector
	value, ok := selectorMap["weblogic.createdByOperator"]
	if !ok || value == "false" {
		return webLogicPorts
	}
	for _, servicePort := range service.Spec.Ports {
		// Check if service port name is one of the predefined WebLogic port names
		for _, webLogicPortName := range weblogicPortNames {
			if servicePort.Name == webLogicPortName {
				webLogicPorts = append(webLogicPorts, servicePort)
			}
		}
	}
	return webLogicPorts
}

// createVirtualServiceMatchURIFromIngressTraitPath create the virtual service match uri map from an ingress trait path
// This is primarily used to setup defaults when either path or type are not present in the ingress path.
// If the provided ingress path doesn't contain a path it is default to /
// If the provided ingress path doesn't contain a type it is defaulted to prefix if path is / and exact otherwise.
func createVirtualServiceMatchURIFromIngressTraitPath(path vzapi.IngressPath) *istio.StringMatch {
	// Default path to /
	p := strings.TrimSpace(path.Path)
	if p == "" {
		p = "/"
	}
	// If path is / default type to prefix
	// If path is not / default to exact
	t := strings.ToLower(strings.TrimSpace(path.PathType))
	if t == "" {
		if p == "/" {
			t = "prefix"
		} else {
			t = "exact"
		}
	}

	switch t {
	case "regex":
		return &istio.StringMatch{MatchType: &istio.StringMatch_Regex{Regex: p}}
	case "prefix":
		return &istio.StringMatch{MatchType: &istio.StringMatch_Prefix{Prefix: p}}
	default:
		return &istio.StringMatch{MatchType: &istio.StringMatch_Exact{Exact: p}}
	}
}

// getPathsFromRule gets the paths from a trait.
// If the trait has no paths a default path is returned.
func getPathsFromRule(rule vzapi.IngressRule) []vzapi.IngressPath {
	paths := rule.Paths
	// If there are no paths create a default.
	if len(paths) == 0 {
		paths = []vzapi.IngressPath{{Path: "/", PathType: "prefix"}}
	}
	return paths
}

// buildDomainNameForWildcard generates a domain name in the format of "<IP>.<wildcard-domain>"
// Get the IP from Istio resources
func buildDomainNameForWildcard(cli client.Reader, suffix string) (string, error) {
	istioIngressGateway := "istio-ingressgateway"
	IstioSystemNamespace := "istio-system"
	istio := corev1.Service{}
	err := cli.Get(context.TODO(), types.NamespacedName{Name: istioIngressGateway, Namespace: IstioSystemNamespace}, &istio)
	if err != nil {
		return "", err
	}
	var IP string
	if istio.Spec.Type == corev1.ServiceTypeLoadBalancer || istio.Spec.Type == corev1.ServiceTypeNodePort {
		if len(istio.Spec.ExternalIPs) > 0 {
			IP = istio.Spec.ExternalIPs[0]
		} else if len(istio.Status.LoadBalancer.Ingress) > 0 {
			IP = istio.Status.LoadBalancer.Ingress[0].IP
		} else {
			return "", fmt.Errorf("%s is missing loadbalancer IP", istioIngressGateway)
		}
	} else {
		return "", fmt.Errorf("unsupported service type %s for istio_ingress", string(istio.Spec.Type))
	}
	domain := IP + "." + suffix
	return domain, nil
}