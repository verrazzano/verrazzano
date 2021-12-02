// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ComponentNamespace is the NGINX namespace for verrazzano
	ComponentNamespace = "ingress-nginx"

	// ValuesFileOverride Name of the values file override for NGINX
	ValuesFileOverride = "ingress-nginx-values.yaml"

	ControllerName = vpoconst.NGINXControllerServiceName
	backendName    = "ingress-controller-ingress-nginx-defaultbackend"
)

func IsReady(context spi.ComponentContext, name string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: ControllerName, Namespace: namespace},
		{Name: backendName, Namespace: namespace},
	}
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1)
}

func AppendOverrides(context spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := context.EffectiveCR()
	ingressType, err := vzconfig.GetServiceType(cr)
	if err != nil {
		return []bom.KeyValue{}, err
	}

	newKvs := append(kvs, bom.KeyValue{Key: "controller.service.type", Value: string(ingressType)})

	if cr.Spec.Components.DNS != nil && cr.Spec.Components.DNS.OCI != nil {
		newKvs = append(newKvs, bom.KeyValue{Key: "controller.service.annotations.external-dns\\.alpha\\.kubernetes\\.io/ttl", Value: "60", SetString: true})
		hostName := fmt.Sprintf("verrazzano-ingress.%s.%s", cr.Spec.EnvironmentName, cr.Spec.Components.DNS.OCI.DNSZoneName)
		newKvs = append(newKvs, bom.KeyValue{Key: "controller.service.annotations.external-dns\\.alpha\\.kubernetes\\.io/hostname", Value: hostName})
	}

	// Convert NGINX install-args to helm overrides
	newKvs = append(newKvs, helm.GetInstallArgs(getInstallArgs(cr))...)
	return newKvs, nil
}

// PreInstall Create and label the NGINX namespace, and create any override helm args needed
func PreInstall(compContext spi.ComponentContext, name string, namespace string, dir string) error {
	if compContext.IsDryRun() {
		compContext.Log().Infof("NGINX PostInstall dry run")
		return nil
	}
	compContext.Log().Infof("Adding label needed by network policies to ingress-nginx namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = "ingress-nginx"
		ns.Labels["istio-injection"] = "enabled"
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// PostInstall Patch the controller service ports based on any user-supplied overrides
func PostInstall(ctx spi.ComponentContext, _ string, _ string) error {
	if ctx.IsDryRun() {
		ctx.Log().Infof("NGINX PostInstall dry run")
		return nil
	}
	// Add any port specs needed to the service after boot
	ingressConfig := ctx.EffectiveCR().Spec.Components.Ingress
	if ingressConfig == nil {
		return nil
	}
	if len(ingressConfig.Ports) == 0 {
		return nil
	}

	c := ctx.Client()
	svcPatch := v1.Service{}
	if err := c.Get(context.TODO(), types.NamespacedName{Name: ControllerName, Namespace: ComponentNamespace}, &svcPatch); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	mergeFromSvc := client.MergeFrom(svcPatch.DeepCopy())
	svcPatch.Spec.Ports = ingressConfig.Ports
	if err := c.Patch(context.TODO(), &svcPatch, mergeFromSvc); err != nil {
		return err
	}
	return nil
}

// getInstallArgs get the install args for NGINX
func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.Ingress == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.Ingress.NGINXInstallArgs
}


// Identify the service type, LB vs NodePort
func GetServiceType(cr *vzapi.Verrazzano) (vzapi.IngressType, error) {
	ingressConfig := cr.Spec.Components.Ingress
	if ingressConfig == nil || len(ingressConfig.Type) == 0 {
		return vzapi.LoadBalancer, nil
	}
	switch ingressConfig.Type {
	case vzapi.NodePort, vzapi.LoadBalancer:
		return ingressConfig.Type, nil
	default:
		return "", fmt.Errorf("Unrecognized ingress type %s", ingressConfig.Type)
	}
}

// GetIngressIP Returns the ingress IP of the LoadBalancer
// - port of install scripts function get_verrazzano_ingress_ip in config.sh
func GetIngressIP(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	// Default for NodePort services
	// - On MAC and Windows, container IP is not accessible.  Port forwarding from 127.0.0.1 to container IP is needed.
	ingressIP := "127.0.0.1"
	serviceType, err := GetServiceType(vz)
	if err != nil {
		return "", err
	}
	if serviceType == vzapi.LoadBalancer || serviceType == vzapi.NodePort {
		svc := v1.Service{}
		if err := client.Get(context.TODO(), types.NamespacedName{Name: ControllerName, Namespace: ComponentNamespace}, &svc); err != nil {
			return "", err
		}
		// Test for IP from status, if that is not present then assume an on premises installation and use the externalIPs hint
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			ingressIP = svc.Status.LoadBalancer.Ingress[0].IP
		} else if len(svc.Spec.ExternalIPs) > 0 {
			// In case of OLCNE, the Status.LoadBalancer.Ingress field will be empty, so use the external IP if present
			ingressIP = svc.Spec.ExternalIPs[0]
		} else {
			return "", fmt.Errorf("No IP found for LoadBalancer service type")
		}
	}
	return ingressIP, nil
}

// GetDNSSuffix Returns the DNS suffix for the Verrazzano installation
// - port of install script function get_dns_suffix from config.sh
func GetDNSSuffix(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	var dnsSuffix string
	dnsConfig := vz.Spec.Components.DNS
	if dnsConfig == nil || dnsConfig.Wildcard != nil {
		ingressIP, err := GetIngressIP(client, vz)
		if err != nil {
			return "", err
		}
		dnsSuffix = fmt.Sprintf("%s.%s", ingressIP, getWildcardDomain(dnsConfig))
	} else if dnsConfig.OCI != nil {
		dnsSuffix = dnsConfig.OCI.DNSZoneName
	} else if dnsConfig.External != nil {
		dnsSuffix = dnsConfig.External.Suffix
	}
	if len(dnsSuffix) == 0 {
		return "", fmt.Errorf("Invalid OCI DNS configuration, no zone name specified")
	}
	return dnsSuffix, nil
}

// BuildDNSDomain Constructs the full DNS subdomain for the deployment
func BuildDNSDomain(client client.Client, vz *vzapi.Verrazzano) (string, error) {
	dnsSuffix, err := GetDNSSuffix(client, vz)
	if err != nil {
		return "", err
	}
	envName := GetEnvName(vz)
	dnsDomain := fmt.Sprintf("%s.%s", envName, dnsSuffix)
	return dnsDomain, nil
}

// GetEnvName Returns the configured environment name, or "default" if not specified in the configuration
func GetEnvName(vz *vzapi.Verrazzano) string {
	envName := vz.Spec.EnvironmentName
	if len(envName) == 0 {
		envName = "default"
	}
	return envName
}

// IsExternalDNSEnabled Indicates if the external-dns service is expected to be deployed, true if OCI DNS is configured
func IsExternalDNSEnabled(vz *vzapi.Verrazzano) bool {
	if vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil {
		return true
	}
	return false
}

// getWildcardDomain Get the wildcard domain from the Verrazzano config
func getWildcardDomain(dnsConfig *vzapi.DNSComponent) string {
	wildcardDomain := "nip.io"
	if dnsConfig != nil && dnsConfig.Wildcard != nil && len(dnsConfig.Wildcard.Domain) > 0 {
		wildcardDomain = dnsConfig.Wildcard.Domain
	}
	return wildcardDomain
}
