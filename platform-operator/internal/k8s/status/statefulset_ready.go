// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulsetReady Check that the named statefulsets have the minimum number of specified replicas ready and available
func StatefulsetReady(log vzlog.VerrazzanoLogger, client client.Client, statefulsets []types.NamespacedName, expectedReplicas int32, prefix string) bool {
	for _, namespacedName := range statefulsets {
		statefulset := appsv1.StatefulSet{}
		if err := client.Get(context.TODO(), namespacedName, &statefulset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for statefulset %v to exist", prefix, namespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Failed getting statefulset %v: %v", namespacedName, err)
			return false
		}
		if statefulset.Status.ReadyReplicas < expectedReplicas {
			log.Progressf("%s is waiting for statefulset %s replicas to be %v. Current replicas is %v", prefix, namespacedName,
				expectedReplicas, statefulset.Status.ReadyReplicas)
			return false
		}
	}
	log.Oncef("%s has enough replicas for statefulsets %v", prefix, statefulsets)
	return true
}
