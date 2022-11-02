// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

// ArgoCD HTTPS Configuration

const (
	// ArgoCDName is the name of the component
	ArgoCDName          = "argocd"
	ArgoCDIngressCAName = "tls-argocd-ingress"
	ArgoCDCACert        = "ca.crt"
	ArgoCDService       = "argocd-server"
)

// ArgoCD deployments
const (
	ArgoCDApplicationSetController = "argocd-applicationset-controller"
	ArgoCDDexServer                = "argocd-dex-server"
	ArgoCDNotificationController   = "argocd-notifications-controller"
	ArgoCDRedis                    = "argocd-redis"
	ArgoCDRepoServer               = "argocd-repo-server"
	ArgoCDServer                   = "argocd-server"
)
