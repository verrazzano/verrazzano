// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginx

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	helm2 "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const (
	// ValuesFileOverride Name of the values file override for NGINX
	ValuesFileOverride = "ingress-nginx-values.yaml"

	ControllerName = vpoconst.NGINXControllerServiceName
	backendName    = "ingress-controller-ingress-nginx-defaultbackend"
)

func (c nginxComponent) isNginxReady(context spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	// Verify that the ingress-nginx service has an external IP before completing post-install
	_, err := vzconfig.GetIngressIP(context.Client(), context.EffectiveCR())
	// Only log the message for if the request comes from this component's context
	// Otherwise, the message is logged for each component that checks the status of the ingress controller
	if err != nil && context.GetComponent() == ComponentName {
		context.Log().Progressf("Ingress external IP pending for component %s: %s", ComponentName, err.Error())
	}
	return err == nil && ready.DeploymentsAreReady(context.Log(), context.Client(), c.AvailabilityObjects.DeploymentNames, 1, prefix)
}

func AppendOverrides(context spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := context.EffectiveCR()
	ingressType, err := vzconfig.GetIngressServiceType(cr)
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
		compContext.Log().Debug("NGINX PostInstall dry run")
		return nil
	}
	compContext.Log().Debug("Adding label needed by network policies to ingress-nginx namespace")
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = "verrazzano-ingress-nginx"
		istio := compContext.EffectiveCR().Spec.Components.Istio
		if istio != nil && istio.IsInjectionEnabled() {
			ns.Labels["istio-injection"] = "enabled"
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// PostInstall Patch the controller service ports based on any user-supplied overrides
func PostInstall(ctx spi.ComponentContext, _ string, _ string) error {
	if ctx.IsDryRun() {
		ctx.Log().Debug("NGINX PostInstall dry run")
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

// GetOverrides gets the install overrides
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Ingress != nil {
			return effectiveCR.Spec.Components.Ingress.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.IngressNGINX != nil {
			return effectiveCR.Spec.Components.IngressNGINX.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}

// DeterminNamespaceForIngressNGINX determines the namespace for Ingress NGINX
func (c nginxComponent) DeterminNamespaceForIngressNGINX(context spi.ComponentContext) (bool, error) {
	const vzClass = "verrazzano-nginx"

	type YamlConfig struct {
		Controller struct {
			IngressClassResource struct {
				Name string `json:"name"`
			}
		}
	}

	// See if NGINX is installed in the ingress-nginx namespace
	found, err := helm2.IsReleaseInstalled(c.ReleaseName, vpoconst.OldIngressNginxNamespace)
	if err != nil {
		context.Log().ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
	}
	if found {
		valMap, err := helm2.GetValuesMap(context.Log(), c.ReleaseName, vpoconst.OldIngressNginxNamespace)
		if err != nil {
			return false, err
		}
		b, err := yaml.Marshal(&valMap)
		if err != nil {
			return false, err
		}
		yml := YamlConfig{}
		if err := yaml.Unmarshal(b, &yml); err != nil {
			return false, err
		}
		if yml.Controller.IngressClassResource.Name == vzClass {
			return true, nil
		}

		return false, nil
	}
	return false, nil
}

// DetermineNamespaceForIngressNGINX determines the namespace for Ingress NGINX
func (c nginxComponent) DetermineNamespaceForIngressNGINX(context spi.ComponentContext) (string, error) {
	// Check if Verrazzano NGINX is installed in the ingress-nginx namespace
	installed, err := c.isNGINXInstalledInOldNamespace(context.Log())
	if err != nil {
		context.Log().ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
	}
	if installed {
		// If Ingress NGINX is already installed ingress-nginx then don't change it.
		// This is to avoid creating a new service in the new namespac, thus causing an
		// LB to be created.
		return vpoconst.OldIngressNginxNamespace, nil
	}

	return vpoconst.IngressNginxNamespace, nil
}

// isNGINXInstalledInOldNamespace determines the namespace for Ingress NGINX
func (c nginxComponent) isNGINXInstalledInOldNamespace(log vzlog.VerrazzanoLogger) (bool, error) {
	const vzClass = "verrazzano-nginx"

	type YamlConfig struct {
		Controller struct {
			IngressClassResource struct {
				Name string `json:"name"`
			}
		}
	}

	// See if NGINX is installed in the ingress-nginx namespace
	found, err := helm2.IsReleaseInstalled(c.ReleaseName, vpoconst.OldIngressNginxNamespace)
	if err != nil {
		log.ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", vpoconst.OldIngressNginxNamespace, c.ReleaseName, err.Error())
	}
	if found {
		valMap, err := helm2.GetValuesMap(log, c.ReleaseName, vpoconst.OldIngressNginxNamespace)
		if err != nil {
			return false, err
		}
		b, err := yaml.Marshal(&valMap)
		if err != nil {
			return false, err
		}
		yml := YamlConfig{}
		if err := yaml.Unmarshal(b, &yml); err != nil {
			return false, err
		}
		if yml.Controller.IngressClassResource.Name == vzClass {
			return true, nil
		}

		return false, nil
	}
	return false, nil
}
