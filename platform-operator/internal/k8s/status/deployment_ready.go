// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package status

import (
	"context"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentsReady Check that the named deployments have the minimum number of specified replicas ready and available
func DeploymentsReady(log *zap.SugaredLogger, client clipkg.Client, deployments []types.NamespacedName, expectedReplicas int32) bool {
	for _, namespacedName := range deployments {
		deployment := appsv1.Deployment{}
		if err := client.Get(context.TODO(), namespacedName, &deployment); err != nil {
			if errors.IsNotFound(err) {
				log.Infof("%v deployment not found", namespacedName)
				// Deployment not found
				return false
			}
			log.Errorf("Unexpected error checking %v status: %v", namespacedName, err)
			return false
		}
		if deployment.Status.AvailableReplicas < expectedReplicas {
			log.Infof("Not enough available replicas for the %v deployment yet", namespacedName)
			return false
		}
	}
	return true
}
