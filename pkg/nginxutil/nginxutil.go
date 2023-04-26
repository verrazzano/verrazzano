// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginxutil

import (
	helm2 "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"sigs.k8s.io/yaml"
)

const helmReleaseName = "ingress-controller"

// This is set by verrazzano controller.go at startup.  It has to be injected
// since is an import cycle if this code uses component.nginx.
var ingressNGINXNamespace = vpoconst.IngressNginxNamespace

// SetIngressNGINXNamespace sets the namespace, this is done at VZ reconcile startup, see controller.go
func SetIngressNGINXNamespace(ns string) {
	ingressNGINXNamespace = ns
}

// IngressNGINXNamespace returns the ingress-nginx namespace
func IngressNGINXNamespace() string {
	return ingressNGINXNamespace
}

// DetermineNamespaceForIngressNGINX determines the namespace for Ingress NGINX
func DetermineNamespaceForIngressNGINX(log vzlog.VerrazzanoLogger) (string, error) {
	// Check if Verrazzano NGINX is installed in the ingress-nginx namespace
	legacy, err := isLegacyNGINXNamespace(log, helmReleaseName, vpoconst.LegacyIngressNginxNamespace)
	if err != nil {
		return "", log.ErrorfNewErr("Failed checking if the old ingress-nginx chart %s/%s is installed: %v", vpoconst.LegacyIngressNginxNamespace, helmReleaseName, err.Error())
	}
	ingressNGINXNamespace = getNamespaceForIngressNGINX(legacy)
	log.Infof("Ingress NGINX namespace is %s", ingressNGINXNamespace)
	return ingressNGINXNamespace, nil
}

// isLegacyNGINXNamespace determines the namespace for Ingress NGINX
func isLegacyNGINXNamespace(log vzlog.VerrazzanoLogger, releaseName string, namespace string) (bool, error) {
	const vzClass = "verrazzano-nginx"

	// Define structs needed to marshal YAML.  Fields must be public
	type IngressClassResource struct {
		Name            string `json:"name"`
		ControllerValue string `json:"controllerValue"`
	}
	type Controller struct {
		IngressClassResource `json:"ingressClassResource"`
	}
	type helmValues struct {
		Controller `json:"controller"`
	}

	// See if NGINX is installed in the ingress-nginx namespace
	found, err := helm2.IsReleaseInstalled(releaseName, namespace)
	if err != nil {
		log.ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", namespace, releaseName, err.Error())
	}
	if found {
		valMap, err := helm2.GetValuesMap(log, releaseName, namespace)
		if err != nil {
			return false, log.ErrorfNewErr("Error getting helm values: %v", err.Error())
		}
		b, err := yaml.Marshal(&valMap)
		if err != nil {
			return false, log.ErrorfNewErr("Error marshaling helm values: %v", err.Error())
		}
		vals := helmValues{}
		if err := yaml.Unmarshal(b, &vals); err != nil {
			return false, log.ErrorfNewErr("Error unmarshaling helm values: %v", err.Error())
		}
		if vals.Controller.IngressClassResource.Name == vzClass {
			return true, nil
		}
	}
	return false, nil
}

func getNamespaceForIngressNGINX(legacy bool) string {
	if legacy {
		// If Ingress NGINX is already installed ingress-nginx then don't change it.
		// This is to avoid creating a new service in the new namespace, thus causing an
		// LB to be created.
		return vpoconst.LegacyIngressNginxNamespace
	}
	return vpoconst.IngressNginxNamespace
}
