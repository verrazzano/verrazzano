// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "weblogic-operator"

const wlsOperatorDeploymentName = ComponentName

// AppendWeblogicOperatorOverrides appends the WKO-specific helm Value overrides.
func AppendWeblogicOperatorOverrides(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	keyValueOverrides := []bom.KeyValue{
		{
			Key:   "serviceAccount",
			Value: "weblogic-operator-sa",
		},
		{
			Key:   "domainNamespaceSelectionStrategy",
			Value: "LabelSelector",
		},
		{
			Key:   "domainNamespaceLabelSelector",
			Value: "verrazzano-managed",
		},
		{
			Key:   "enableClusterRoleBinding",
			Value: "true",
		},
	}

	kvs = append(kvs, keyValueOverrides...)

	return kvs, nil
}

func WeblogicOperatorPreInstall(ctx spi.ComponentContext, _ string, namespace string, _ string) error {
	var serviceAccount corev1.ServiceAccount
	const accountName = "weblogic-operator-sa"
	c := ctx.Client()
	if err := c.Get(context.TODO(), types.NamespacedName{Name: accountName, Namespace: namespace}, &serviceAccount); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	serviceAccount = corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: namespace,
		},
	}
	if err := c.Create(context.TODO(), &serviceAccount); err != nil {
		if errors.IsAlreadyExists(err) {
			// Sometimes we get this, not an error it already exist.
			return nil
		}
		return err
	}
	return nil
}

func IsWeblogicOperatorReady(ctx spi.ComponentContext, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: wlsOperatorDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1)
}
