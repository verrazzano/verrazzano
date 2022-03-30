// Copyright (c) 2022, Oracle and/or its affiliates.
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

// DaemonSetsReady Check that the named daemonsets have the minimum number of specified nodes ready and available
func DaemonSetsReady(log vzlog.VerrazzanoLogger, client client.Client, daemonsets []types.NamespacedName, expectedNodes int32, prefix string) bool {
	for _, namespacedName := range daemonsets {
		daemonset := appsv1.DaemonSet{}
		if err := client.Get(context.TODO(), namespacedName, &daemonset); err != nil {
			if errors.IsNotFound(err) {
				log.Progressf("%s is waiting for daemonsets %v to exist", prefix, namespacedName)
				// StatefulSet not found
				return false
			}
			log.Errorf("Failed getting daemonset %v: %v", namespacedName, err)
			return false
		}
		if daemonset.Status.NumberAvailable < expectedNodes {
			log.Progressf("%s is waiting for daemonset %s nodes to be %v. Current nodes is %v", prefix, namespacedName,
				expectedNodes, daemonset.Status.NumberAvailable)
			return false
		}
	}
	log.Oncef("%s has enough nodes for daemonsets %v", prefix, daemonsets)
	return true
}
