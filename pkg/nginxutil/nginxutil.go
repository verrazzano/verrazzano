// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package nginxutil

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/namespace"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
)

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
	legacyNSExists, err := namespace.CheckIfVerrazzanoManagedNamespaceExists(vpoconst.LegacyIngressNginxNamespace)
	if err != nil {
		return "", log.ErrorfNewErr("Failed checking for legacy Ingress NGINX namespace %s: %v", vpoconst.LegacyIngressNginxNamespace, err.Error())
	}
	ingressNGINXNamespace = getNamespaceForIngressNGINX(legacyNSExists)
	log.Oncef("Ingress NGINX namespace is %s", ingressNGINXNamespace)
	return ingressNGINXNamespace, nil
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
