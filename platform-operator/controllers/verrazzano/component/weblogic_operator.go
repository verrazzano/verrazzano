// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"context"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// appendWeblogicOperatorOverrides appends the WKO-specific helm value overrides.
func appendWeblogicOperatorOverrides(_ *zap.SugaredLogger, releaseName string, _ string, _ string, kvs []keyValue) ([]keyValue, error) {
	keyValueOverrides := []keyValue{
		{
			key:   "serviceAccount",
			value: "weblogic-operator-sa",
		},
		{
			key:   "domainNamespaceSelectionStrategy",
			value: "LabelSelector",
		},
		{
			key:   "domainNamespaceLabelSelector",
			value: "verrazzano-managed",
		},
		{
			key:   "enableClusterRoleBinding",
			value: "true",
		},
	}

	kvs = append(kvs, keyValueOverrides...)

	return kvs, nil
}

func weblogicOperatorPreInstall(log *zap.SugaredLogger, client clipkg.Client, _ string, namespace string, _ string) ([]keyValue, error) {
	var serviceAccount corev1.ServiceAccount
	const accountName = "weblogic-operator-sa"
	if err := client.Get(context.TODO(), types.NamespacedName{Name: accountName, Namespace: namespace}, &serviceAccount); err != nil {
		if errors.IsAlreadyExists(err) {
			// Service account already exists in the target namespace
			return []keyValue{}, nil
		}
		if !errors.IsNotFound(err) {
			// Unexpected error
			return []keyValue{}, err
		}
	}
	serviceAccount = corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      accountName,
			Namespace: namespace,
		},
	}
	if err := client.Create(context.TODO(), &serviceAccount); err != nil {
		return []keyValue{}, err
	}
	return []keyValue{}, nil
}
