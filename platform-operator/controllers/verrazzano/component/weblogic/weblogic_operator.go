// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "weblogic-operator"

const wlsOperatorDeploymentName = ComponentName

// AppendWeblogicOperatorOverrides appends the WKO-specific helm Value overrides.
func AppendWeblogicOperatorOverrides(_ *zap.SugaredLogger, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
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

func WeblogicOperatorPreInstall(log *zap.SugaredLogger, client clipkg.Client, _ string, namespace string, _ string) ([]bom.KeyValue, error) {
	var serviceAccount corev1.ServiceAccount
	const accountName = "weblogic-operator-sa"
	if err := client.Get(context.TODO(), types.NamespacedName{Name: accountName, Namespace: namespace}, &serviceAccount); err != nil {
		if errors.IsAlreadyExists(err) {
			// Service account already exists in the target namespace
			return []bom.KeyValue{}, nil
		}
		if !errors.IsNotFound(err) {
			// Unexpected error
			return []bom.KeyValue{}, err
		}
	}
	serviceAccount = corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: namespace,
		},
	}
	if err := client.Create(context.TODO(), &serviceAccount); err != nil {
		return []bom.KeyValue{}, err
	}
	return []bom.KeyValue{}, nil
}

func IsWeblogicOperatorReady(log *zap.SugaredLogger, c clipkg.Client, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: wlsOperatorDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(log, c, deployments, 1)
}
