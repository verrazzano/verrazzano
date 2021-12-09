// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulsetReady Check that the named statefulsets have the minimum number of specified replicas ready and available
func StatefulsetReady(log *zap.SugaredLogger, client client.Client, statefulsets []types.NamespacedName, expectedReplicas int32) bool {
	for _, namespacedName := range statefulsets {
		statefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), namespacedName, &statefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Infof("Keycloak StatefulSetsReady: %v statefulSet not found", namespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Keycloak StatefulSetsReady: Unexpected error checking %v status: %v", namespacedName, err)
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Infof("Keycloak StatefulSetsReady: Not enough available replicas for the %v statefulset yet", namespacedName)
			return false
		}
	}
	return true
}
