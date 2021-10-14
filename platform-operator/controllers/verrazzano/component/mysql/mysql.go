// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/bom"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const (
	ComponentName = "mysql"
	secretName    = "mysql"
	helmPwd       = "mysqlPassword"
	helmRootPwd   = "mysqlRootPassword"
	mysqlKey      = "mysql-password"
	mysqlRootKey  = "mysql-root-password"
)

func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	secret := &corev1.Secret{}
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
	return kvs, nil
}
