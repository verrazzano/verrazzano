// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginxutil

import (
	helm2 "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GetIngressNGINXNamespace get the Ingress NGINX namespace from the metadata annotation
func GetIngressNGINXNamespace(meta metav1.ObjectMeta) string {
	anno := meta.Annotations
	if anno == nil {
		return vpoconst.IngressNginxNamespace
	}
	val, ok := anno[vpoconst.IngressNginxNamespaceAnnotation]
	if ok {
		return val
	}
	return vpoconst.IngressNginxNamespace
}

// EnsureAnnotationForIngressNGINX ensures that the VZ CR has an annotation for Ingress NGINX namespace
// Return true if VZ update needed.
func EnsureAnnotationForIngressNGINX(log vzlog.VerrazzanoLogger, meta *metav1.ObjectMeta) (bool, error) {
	anno := meta.Annotations
	if anno == nil {
		anno = make(map[string]string)
	}
	_, ok := anno[vpoconst.IngressNginxNamespaceAnnotation]
	if ok {
		return false, nil
	}
	name, err := DetermineNamespaceForIngressNGINX(log)
	if err != nil {
		return false, err
	}
	anno[vpoconst.IngressNginxNamespaceAnnotation] = name
	return true, nil
}

// DetermineNamespaceForIngressNGINX determines the namespace for Ingress NGINX
func DetermineNamespaceForIngressNGINX(log vzlog.VerrazzanoLogger) (string, error) {
	// Check if Verrazzano NGINX is installed in the ingress-nginx namespace
	installed, err := isVzNGINXInstalledInNamespace(log, helmReleaseName, vpoconst.LegacyIngressNginxNamespace)
	if err != nil {
		log.ErrorfNewErr("Failed checking if the old ingress-nginx chart %s/%s is installed: %v", vpoconst.LegacyIngressNginxNamespace, helmReleaseName, err.Error())
	}
	if installed {
		// If Ingress NGINX is already installed ingress-nginx then don't change it.
		// This is to avoid creating a new service in the new namespace, thus causing an
		// LB to be created.
		return vpoconst.LegacyIngressNginxNamespace, nil
	}

	return vpoconst.IngressNginxNamespace, nil
}

// isNGINXInstalledInOldNamespace determines the namespace for Ingress NGINX
func isVzNGINXInstalledInNamespace(log vzlog.VerrazzanoLogger, releaseName string, namespace string) (bool, error) {
	const vzClass = "verrazzano-nginx"

	type YamlConfig struct {
		Controller struct {
			IngressClassResource struct {
				Name string `json:"name"`
			}
		}
	}

	// See if NGINX is installed in the ingress-nginx namespace
	found, err := helm2.IsReleaseInstalled(releaseName, namespace)
	if err != nil {
		log.ErrorfNewErr("Error checking if the old ingress-nginx chart %s/%s is installed error: %v", namespace, releaseName, err.Error())
	}
	if found {
		valMap, err := helm2.GetValuesMap(log, releaseName, namespace)
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
	}
	return false, nil
}
