// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd

import (
	"context"
	"fmt"
	"strings"
	"time"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

const  (
	adminPolicy = `g, verrazzano-admins, role:admin`
	policyCSV = `policy.csv`
)

type OIDCConfig struct {
	Name            string   `json:"name"`
	Issuer          string   `json:"issuer"`
	ClientID        string   `json:"clientID"`
	ClientSecret    string   `json:"clientSecret"`
	RequestedScopes []string `json:"requestedScopes"`
	RootCA          string   `json:"rootCA"`
}

// patchArgoCDSecret
func patchArgoCDSecret(component argoCDComponent, ctx spi.ComponentContext) error {
	clientSecret, err := component.ArgoClientSecretProvider.GetClientSecret(ctx)
	if err != nil {
		ctx.Log().ErrorfNewErr("failed configuring keycloak as OIDC provider for Argo CD, unable to fetch Argo CD client secret: %s", err)
		return err
	}

	// update the secret with the updated client secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "argocd-secret",
			Namespace: constants.ArgoCDNamespace,
		},
	}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data["oidc.keycloak.clientSecret"] = []byte(clientSecret)

		return nil
	}); err != nil {
		ctx.Log().ErrorfNewErr("Failed to patch the Argo CD secret argocd-secret: %s", err)
		return err
	}

	ctx.Log().Debugf("patchArgoCDSecret: Argo CD secret operation result: %v", err)
	return nil
}

// patchArgoCDConfigMap
func patchArgoCDConfigMap(ctx spi.ComponentContext) error {
	c := ctx.Client()
	keycloakIngress, err := k8sutil.GetURLForIngress(c, "keycloak", "keycloak", "https")
	if err != nil {
		ctx.Log().ErrorfNewErr("failed getting the Keycloak Ingress URL: %s", err)
		return err
	}

	argoCDURL, err := k8sutil.GetURLForIngress(c, "argocd-server", "argocd", "https")
	if err != nil {
		ctx.Log().ErrorfNewErr("failed getting the Argo CD Ingress URL: %s", err)
		return err
	}

	keycloakURL := fmt.Sprintf("%s/%s", keycloakIngress, "auth/realms/verrazzano-system")

	ctx.Log().Debugf("Getting ArgoCD TLS root CA")
	caCert, err := GetRootCA(ctx)
	if err != nil {
		ctx.Log().ErrorfNewErr("Failed to get Argo CD TLS root CA: %v", err)
		return err
	}

	conf := &OIDCConfig{
		Name:         "Keycloak",
		Issuer:       keycloakURL,
		ClientID:     "argocd",
		ClientSecret: "$oidc.keycloak.clientSecret",
		RequestedScopes: []string{
			"openid",
			"profile",
			"email",
			"groups",
		},
		RootCA: string(caCert),
	}

	data, err := yaml.Marshal(conf)
	if err != nil {
		fmt.Println(err)
		return err
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ArgoCDCM,
			Namespace: constants.ArgoCDNamespace,
		},
	}

	// Add the oidc configuration to enable our keycloak authentication.
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), cm, func() error {
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data["url"] = argoCDURL
		cm.Data["oidc.config"] = string(data)

		return nil
	}); err != nil {
		ctx.Log().ErrorfNewErr("Failed to patch the Argo CD configmap argocd-cm: %s", err)
		return err
	}

	ctx.Log().Debugf("patchArgoCDConfigMap: Argo CD cm operation result: %v", err)
	return nil
}

// patchArgoCDRbacConfigMap
func patchArgoCDRbacConfigMap(ctx spi.ComponentContext) error {
	rbaccm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.ArgoCDRBACCM,
			Namespace: constants.ArgoCDNamespace,
		},
	}

	var err error

	// Disable the built-in admin user. Grant admin (role:admin) to verrazzano-admins group
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), rbaccm, func() error {
		if rbaccm.Data == nil {
			rbaccm.Data = make(map[string]string)
		}
		// Make sure the policy.csv has the verrazzano admin policy
		policy, ok := rbaccm.Data[policyCSV]
		if !ok || len(policy) == 0 {
			// There is no policy.csv override, Add the verrazzano admin
			rbaccm.Data[policyCSV] = adminPolicy
		} else if !strings.Contains(policy, adminPolicy) {
			// The policy.csv exists, but doesn't have the verrazzano admin policy.  Add it.
			trim := strings.TrimSpace(policy)
			s := fmt.Sprintf("%s\n%s", trim, adminPolicy)
			rbaccm.Data[policyCSV] = s
		}
		return nil
	}); err != nil {
		ctx.Log().ErrorfNewErr("Failed to patch the Argo CD configmap argocd-rbac-cm: %s", err)
		return err
	}

	ctx.Log().Debugf("patchArgoCDRbacConfigMap: Argo CD rbac cm operation result: %v", err)
	return nil
}

// restartArgoCDServerDeploy restarts the argocd server deployment
func restartArgoCDServerDeploy(ctx spi.ComponentContext) error {
	deployment := &appsv1.Deployment{}
	deployName := types.NamespacedName{
		Namespace: constants.ArgoCDNamespace,
		Name:      common.ArgoCDServer}

	if err := ctx.Client().Get(context.TODO(), deployName, deployment); err != nil {
		return err
	}

	time := time.Now()
	// Annotate the deployment to do a restart of the pods
	deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = buildRestartAnnotationString(time)
	if err := ctx.Client().Update(context.TODO(), deployment); err != nil {
		ctx.Log().ErrorfNewErr("Failed, error updating Deployment %s annotation to force a pod restart", deployment.Name)
		return err
	}

	return nil
}

// GetRootCA gets the root CA certificate from the Argo CD TLS secret. If the secret does not exist, we
// return a nil slice.
func GetRootCA(ctx spi.ComponentContext) ([]byte, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.ArgoCDNamespace,
		Name:      common.ArgoCDIngressCAName}

	if err := ctx.Client().Get(context.TODO(), nsName, secret); err != nil {
		return nil, err
	}

	return secret.Data[common.ArgoCDCACert], nil
}

// buildRestartAnnotationString returns the current time for annotating deployment to restart the pod
func buildRestartAnnotationString(time time.Time) string {
	return time.String()
}
