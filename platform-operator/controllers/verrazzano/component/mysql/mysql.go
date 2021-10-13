// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const (
	ComponentName = "mysql"
	SecretName    = "mysql"
	HelmPwd       = "mysqlPassword"
	HelmRootPwd   = "mysqlRootPassword"
)

func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	cr := compContext.EffectiveCR()
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: vzconst.KeycloakNamespace,
		Name:      SecretName}

	if err := compContext.Client().Get(context.TODO(), nsName, secret); err != nil {
		return []bom.KeyValue{}, err
	}

	// Force mysql to use the initial password and root password during the upgrade, by specifying as helm overrides
	kvs = append(kvs, bom.KeyValue{
		Key:   HelmPwd,
		Value: string(secret.Data["mysql-password"]),
	})
	kvs = append(kvs, bom.KeyValue{
		Key:   HelmRootPwd,
		Value: string(secret.Data["mysql-root-password"]),
	})
	newKvs := append(kvs, helm.GetInstallArgs(getInstallArgs(cr))...)
	return newKvs, nil
}

func getInstallArgs(cr *vzapi.Verrazzano) []vzapi.InstallArgs {
	if cr.Spec.Components.MySQL == nil {
		return []vzapi.InstallArgs{}
	}
	return cr.Spec.Components.MySQL.MySQLInstallArgs
}
