// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const (
	secretName    = "mysql"
	mysqlUsername = "keycloak"
	helmPwd       = "mysqlPassword"
	helmRootPwd   = "mysqlRootPassword"
	mysqlKey      = "mysql-password"
	mysqlRootKey  = "mysql-root-password"
)

func IsReady(context spi.ComponentContext, name string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: name, Namespace: namespace},
	}
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1)
}

// AppendMySQLOverrides appends the the password for database user and root user.
func AppendMySQLOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()
	secret := &v1.Secret{}
	nsName := types.NamespacedName{
		Namespace: vzconst.KeycloakNamespace,
		Name:      secretName}

	if err := compContext.Client().Get(context.TODO(), nsName, secret); err != nil {
		return []bom.KeyValue{}, err
	}

	// Force mysql to use the initial password and root password during the upgrade, by specifying as helm overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   helmPwd,
		Value: string(secret.Data[mysqlKey]),
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   helmRootPwd,
		Value: string(secret.Data[mysqlRootKey]),
	})

	kvs = append(kvs, bom.KeyValue{Key: "mysqlUser", Value: mysqlUsername})

	// Convert NGINX install-args to helm overrides
	kvs = append(kvs, helm.GetInstallArgs(getInstallArgs(cr))...)

	return kvs, nil
}

// PreInstall Create and label the NGINX namespace, and create any override helm args needed
func PreInstall(compContext spi.ComponentContext, name string, namespace string, dir string) error {
	if compContext.IsDryRun() {
		compContext.Log().Infof("MySQL PostInstall dry run")
		return nil
	}
	compContext.Log().Infof("Adding label needed by network policies to %s namespace", namespace)
	ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), compContext.Client(), &ns, func() error {
		if ns.Labels == nil {
			ns.Labels = make(map[string]string)
		}
		ns.Labels["verrazzano.io/namespace"] = namespace
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
