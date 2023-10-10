// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClusterClient creates a controller-runtime client for a given CAPI cluster reference, if the kubeconfig for that cluster is available.
func GetClusterClient(ctx context.Context, cli clipkg.Client, cluster types.NamespacedName, scheme *runtime.Scheme) (clipkg.Client, error) {
	kubeconfigSecret := &corev1.Secret{}
	kubeconfigSecretNSN := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      fmt.Sprintf("%s-kubeconfig", cluster.Name),
	}
	if err := cli.Get(ctx, kubeconfigSecretNSN, kubeconfigSecret); err != nil {
		return nil, err
	}
	kubeconfig, ok := kubeconfigSecret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("no kubeconfig found for cluster %s/%s", cluster.Namespace, cluster.Name)
	}
	config, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return clipkg.New(config, clipkg.Options{
		Scheme: scheme,
	})
}
