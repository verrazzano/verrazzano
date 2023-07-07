// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ArgoCD HTTPS Configuration
const (
	ArgoCDCompName      = "argoCD"
	ArgoCDName          = "argocd"
	ArgoCDIngressCAName = "tls-argocd-ingress"
	ArgoCDCACert        = "ca.crt"
	ArgoCDService       = "argocd-server"
	ArgoCDCM            = "argocd-cm"
	ArgoCDRBACCM        = "argocd-rbac-cm"
)

// ArgoCD deployments
const (
	ArgoCDApplicationSetController = "argocd-applicationset-controller"
	ArgoCDNotificationController   = "argocd-notifications-controller"
	ArgoCDRedis                    = "argocd-redis"
	ArgoCDRepoServer               = "argocd-repo-server"
	ArgoCDServer                   = "argocd-server"
)

// ArgoCD GVK
const (
	ArgoCDAPIGroup           = "argoproj.io"
	ArgoCDKindApplication    = "Application"
	ArgoCDKindAppProject     = "AppProject"
	ArgoCDKindApplicationSet = "ApplicationSet"
	ArgoCDAPIVersion         = "v1alpha1"
)

// func GetArgoProjAPIGVRForResource returns a argoproj.io/v1alpha1 GroupVersionResource structure for specified kind
func GetArgoProjAPIGVRForResource(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   ArgoCDAPIGroup,
		Version: ArgoCDAPIVersion,
		Kind:    kind,
	}
}
