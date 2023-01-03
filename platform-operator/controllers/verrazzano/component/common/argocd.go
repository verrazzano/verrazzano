// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

// ArgoCD HTTPS Configuration

const (
	// ArgoCDName is the name of the component
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

// auth config
const (
	KeyCloakOIDCConfig = "clientID: argocd\nclientSecret: $oidc.keycloak.clientSecret\nissuer: https://keycloak/auth/realms/verrazzano-system\nname: Keycloak\nrequestedScopes:\n- openid\n- profile\n- email\n- groups\nrootCA: test-ca-argocd\n"
)
