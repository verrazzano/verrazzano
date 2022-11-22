// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"context"
	"fmt"
	"time"

	k8sready "github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Last time the MySQL StatefulSet was ready
var lastTimeStatefulSetReady time.Time

// The start of the timer for determining if an IC object is stuck terminating
var initialTimeICUninstallChecked time.Time

// repairICStuckDeleting - temporary workaround to repair issue where a InnoDBCluster object
// can be stuck terminating (e.g. during uninstall).  The workaround is to recycle the mysql-operator
func (c mysqlComponent) repairICStuckDeleting(ctx spi.ComponentContext) error {
	// Get the IC object
	innoDBCluster, err := getInnoDBCluster(ctx)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if innoDBCluster.GetDeletionTimestamp() == nil {
		*c.InitialTimeICUninstallChecked = time.Time{}
		return nil
	}

	// Found an IC object with a deletion timestamp. Start a timer if this is the first time.
	if c.InitialTimeICUninstallChecked.IsZero() {
		*c.InitialTimeICUninstallChecked = time.Now()
		return fmt.Errorf("Starting check to insure the InnoDBCluster %s/%s is not stuck deleting", ComponentNamespace, helmReleaseName)
	}

	// Initiate repair only if time to wait period has been exceeded
	expiredTime := c.InitialTimeICUninstallChecked.Add(5 * time.Minute)
	if time.Now().After(expiredTime) {
		// Restart the mysql-operator to see if it will finish deleting the IC object
		ctx.Log().Info("Restarting the mysql-operator to see if it will repair InnoDBCluster stuck deleting")

		operPod, err := getMySQLOperatorPod(ctx.Log(), ctx.Client())
		if err != nil {
			return fmt.Errorf("Failed restarting the mysql-operator to repair InnoDBCluster stuck deleting: %v", err)
		}

		if err = ctx.Client().Delete(context.TODO(), operPod, &clipkg.DeleteOptions{}); err != nil {
			return err
		}

		// Clear the timer
		*c.InitialTimeICUninstallChecked = time.Time{}
		return nil
	}

	ctx.Log().Progressf("Waiting for InnoDBCluster %s/%s to be deleted", ComponentNamespace, helmReleaseName)

	return fmt.Errorf("Waiting for InnoDBCluster %s/%s to be deleted", ComponentNamespace, helmReleaseName)
}

// repairMySQLPodsWaitingReadinessGates - temporary workaround to repair issue were a MySQL pod
// can be stuck waiting for its readiness gates to be met.
func (c mysqlComponent) repairMySQLPodsWaitingReadinessGates(ctx spi.ComponentContext) error {
	podsWaiting, err := c.mySQLPodsWaitingForReadinessGates(ctx)
	if err != nil {
		return err
	}
	if podsWaiting {
		// Restart the mysql-operator to see if it will finish setting the readiness gates
		ctx.Log().Info("Restarting the mysql-operator to see if it will repair MySQL pods stuck waiting for readiness gates")

		operPod, err := getMySQLOperatorPod(ctx.Log(), ctx.Client())
		if err != nil {
			return fmt.Errorf("Failed restarting the mysql-operator to repair stuck MySQL pods: %v", err)
		}

		if err = ctx.Client().Delete(context.TODO(), operPod, &clipkg.DeleteOptions{}); err != nil {
			return err
		}

		// Clear the timer
		*c.LastTimeReadinessGateRepairStarted = time.Time{}
	}
	return nil
}

// mySQLPodsWaitingForReadinessGates - detect if there are MySQL pods stuck waiting for
// their readiness gates to be true.
func (c mysqlComponent) mySQLPodsWaitingForReadinessGates(ctx spi.ComponentContext) (bool, error) {
	if c.LastTimeReadinessGateRepairStarted.IsZero() {
		*c.LastTimeReadinessGateRepairStarted = time.Now()
		return false, nil
	}

	// Initiate repair only if time to wait period has been exceeded
	expiredTime := c.LastTimeReadinessGateRepairStarted.Add(5 * time.Minute)
	if time.Now().After(expiredTime) {
		// Check if the current not ready state is due to readiness gates not met
		ctx.Log().Debug("Checking if MySQL not ready due to pods waiting for readiness gates")

		selector := metav1.LabelSelectorRequirement{Key: mySQLComponentLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{mySQLDComponentName}}
		podList := k8sready.GetPodsList(ctx.Log(), ctx.Client(), types.NamespacedName{Namespace: ComponentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selector}})
		if podList == nil || len(podList.Items) == 0 {
			return false, fmt.Errorf("Failed checking MySQL readiness gates, no pods found matching selector %s", selector.String())
		}

		for i := range podList.Items {
			pod := podList.Items[i]
			// Check if the readiness conditions have been met
			conditions := pod.Status.Conditions
			if len(conditions) == 0 {
				return false, fmt.Errorf("Failed checking MySQL readiness gates, no status conditions found for pod %s/%s", pod.Namespace, pod.Name)
			}
			readyCount := 0
			for _, condition := range conditions {
				for _, gate := range pod.Spec.ReadinessGates {
					if condition.Type == gate.ConditionType && condition.Status == v1.ConditionTrue {
						readyCount++
						continue
					}
				}
			}

			// All readiness gates must be true
			if len(pod.Spec.ReadinessGates) != readyCount {
				return true, nil
			}
		}
	}
	return false, nil
}

// getMySQLOperatorPod - return the mysql-operator pod
func getMySQLOperatorPod(log vzlog.VerrazzanoLogger, client clipkg.Client) (*v1.Pod, error) {
	operSelector := metav1.LabelSelectorRequirement{Key: "name", Operator: metav1.LabelSelectorOpIn, Values: []string{mysqloperator.ComponentName}}
	operPodList := k8sready.GetPodsList(log, client, types.NamespacedName{Namespace: mysqloperator.ComponentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{operSelector}})
	if operPodList == nil || len(operPodList.Items) != 1 {
		return nil, fmt.Errorf("no pods found matching selector %s", operSelector.String())
	}
	return &operPodList.Items[0], nil
}

// getInnoDBCluster - get the InnoDBCluster object
func getInnoDBCluster(ctx spi.ComponentContext) (*unstructured.Unstructured, error) {
	innoDBCluster := unstructured.Unstructured{}
	innoDBCluster.SetGroupVersionKind(innoDBClusterGVK)

	// The InnoDBCluster resource name is the helm release name
	nsn := types.NamespacedName{Namespace: ComponentNamespace, Name: helmReleaseName}
	if err := ctx.Client().Get(context.Background(), nsn, &innoDBCluster); err != nil {
		return nil, err
	}
	return &innoDBCluster, nil
}
