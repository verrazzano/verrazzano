// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

// Reusable code for Quick Create controllers

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// ApplyResources applies any number of template files, using the cr as parameter data
func ApplyResources(ctx context.Context, cli clipkg.Client, cr any, files ...string) error {
	y := k8sutil.NewYAMLApplier(cli, "")
	for _, f := range files {
		if err := y.ApplyFT(f, cr); err != nil {
			return err
		}
	}
	return nil
}

func RequeueDelay() ctrl.Result {
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 30 * time.Second,
	}
}

func GetClusterClient(ctx context.Context, adminCli clipkg.Client, clusterRef types.NamespacedName, scheme *runtime.Scheme) (clipkg.Client, error) {
	clusterKubeconfigRef := types.NamespacedName{
		Namespace: clusterRef.Namespace,
		Name:      fmt.Sprintf("%s-kubeconfig", clusterRef.Name),
	}
	kubeconfigSecret := &v1.Secret{}
	if err := adminCli.Get(ctx, clusterKubeconfigRef, kubeconfigSecret); err != nil {
		return nil, err
	}
	kubeconfigBytes, ok := kubeconfigSecret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("no kubeconfig found for cluster %s/%s", clusterRef.Namespace, clusterRef.Name)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, err
	}
	return clipkg.New(restConfig, clipkg.Options{
		Scheme: scheme,
	})
}
