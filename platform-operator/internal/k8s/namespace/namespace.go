// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package namespace

import (
	"context"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Add or replaces the namespace labels with the specified labels
func AddLabels(log *zap.SugaredLogger, client clipkg.Client, ns string, labels map[string]string) error {
	namespace := corev1.Namespace{}
	namespace.Name = ns
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &namespace, func() error {
		if namespace.ObjectMeta.Labels == nil {
			namespace.ObjectMeta.Labels = make(map[string]string)
		}
		for key, _ := range labels {
			namespace.ObjectMeta.Labels[key] = labels[key]
		}
		return nil
	})
	return err
}

// Create a namespace if it doesn't exist
func EnsureExists(log *zap.SugaredLogger, client clipkg.Client, ns string) error {
	namespace := corev1.Namespace{}
	namespace.Name = ns
	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &namespace, func() error {
		return nil
	})
	return err
}
