// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"context"
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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
		{
			Key:   "istioLocalhostBindingsEnabled",
			Value: "false",
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

func isWeblogicOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// GetOverrides returns install overrides for a component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.WebLogicOperator != nil {
			return effectiveCR.Spec.Components.WebLogicOperator.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.WebLogicOperator != nil {
			return effectiveCR.Spec.Components.WebLogicOperator.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
