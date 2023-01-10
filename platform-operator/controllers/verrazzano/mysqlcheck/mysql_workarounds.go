// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"context"
	"fmt"
	"time"

	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	k8sready "github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/mysqloperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mySQLComponentLabel = "component"
	mySQLDComponentName = "mysqld"
	helmReleaseName     = "mysql"
	componentNamespace  = "keycloak"
	componentName       = "mysql"
	repairTimeoutPeriod = 2 * time.Minute
)

var (
	innoDBClusterGVK = schema.GroupVersionKind{
		Group:   "mysql.oracle.com",
		Version: "v2",
		Kind:    "InnoDBCluster",
	}

	// The start of the timer for determining if an IC object is stuck terminating
	initialTimeICUninstallChecked time.Time

	// The start of the timer for determining if any MySQL pods are stuck terminating
	initialTimeMySQLPodsStuckChecked time.Time

	// The start of the timer for determining if any MySQL pods are waiting for readiness gates
	initialTimeReadinessGateChecked time.Time
)

// resetInitialTimeICUninstallChecked allocates an empty time struct
func resetInitialTimeICUninstallChecked() {
	initialTimeICUninstallChecked = time.Time{}
}

// getInitialTimeICUninstallChecked returns the time struct
func getInitialTimeICUninstallChecked() time.Time {
	return initialTimeICUninstallChecked
}

// setInitialTimeICUninstallChecked sets the time struct
func setInitialTimeICUninstallChecked(time time.Time) {
	initialTimeICUninstallChecked = time
}

// resetInitialTimeMySQLPodsStuckChecked allocates an empty time struct
func resetInitialTimeMySQLPodsStuckChecked() {
	initialTimeMySQLPodsStuckChecked = time.Time{}
}

// getInitialTimeMySQLPodsStuckChecked returns the time struct
func getInitialTimeMySQLPodsStuckChecked() time.Time {
	return initialTimeMySQLPodsStuckChecked
}

// setInitialTimeMySQLPodsStuckChecked sets the time struct
func setInitialTimeMySQLPodsStuckChecked(time time.Time) {
	initialTimeMySQLPodsStuckChecked = time
}

// getLastTimeReadinessGateChecked returns the time struct
func getLastTimeReadinessGateChecked() time.Time {
	return initialTimeReadinessGateChecked
}

// setInitialTimeReadinessGateChecked sets the time struct
func setInitialTimeReadinessGateChecked(time time.Time) {
	initialTimeReadinessGateChecked = time
}

// resetInitialTimeReadinessGateChecked sets the time struct
func resetInitialTimeReadinessGateChecked() {
	initialTimeReadinessGateChecked = time.Time{}
}

// RepairICStuckDeleting - temporary workaround to repair issue where a InnoDBCluster object
// can be stuck terminating (e.g. during uninstall).  The workaround is to recycle the mysql-operator.
func RepairICStuckDeleting(ctx spi.ComponentContext) error {
	// Get the IC object
	innoDBCluster, err := getInnoDBCluster(ctx)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if innoDBCluster.GetDeletionTimestamp() == nil {
		resetInitialTimeICUninstallChecked()
		return nil
	}

	// Found an IC object with a deletion timestamp. Start a timer if this is the first time.
	if getInitialTimeICUninstallChecked().IsZero() {
		setInitialTimeICUninstallChecked(time.Now())
		ctx.Log().Progressf("Starting check to insure the InnoDBCluster %s/%s is not stuck deleting", componentNamespace, helmReleaseName)
		return ctrlerrors.RetryableError{}
	}

	// Initiate repair only if time to wait period has been exceeded
	expiredTime := getInitialTimeICUninstallChecked().Add(repairTimeoutPeriod)
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
		resetInitialTimeICUninstallChecked()
		return nil
	}

	ctx.Log().Progressf("Waiting for InnoDBCluster %s/%s to be deleted", componentNamespace, helmReleaseName)

	return ctrlerrors.RetryableError{}
}

// RepairMySQLPodsWaitingReadinessGates - temporary workaround to repair issue were a MySQL pod
// can be stuck waiting for its readiness gates to be met.  The workaround is to recycle the mysql-operator.
func (mc *MySQLChecker) RepairMySQLPodsWaitingReadinessGates() error {
	podsWaiting, err := isPodsWaitingForReadinessGates(mc.log, mc.client)
	if err != nil {
		return err
	}
	if podsWaiting {
		// Start a timer the first time pods are waiting for readiness gates
		if getLastTimeReadinessGateChecked().IsZero() {
			setInitialTimeReadinessGateChecked(time.Now())
			return nil
		}

		// Initiate repair only if time to wait period has been exceeded
		expiredTime := getLastTimeReadinessGateChecked().Add(mc.RepairTimeout)
		if time.Now().After(expiredTime) {
			return restartMySQLOperator(mc.log, mc.client, "MySQL pods waiting for readiness gates")
		}
	}

	// Clear the timer when no pods are waiting
	resetInitialTimeReadinessGateChecked()
	return nil
}

func isPodsWaitingForReadinessGates(log vzlog.VerrazzanoLogger, client clipkg.Client) (bool, error) {
	log.Debug("Checking if MySQL pods waiting for readiness gates")

	selector := metav1.LabelSelectorRequirement{Key: mySQLComponentLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{mySQLDComponentName}}
	podList := k8sready.GetPodsList(log, client, types.NamespacedName{Namespace: componentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selector}})
	if podList == nil || len(podList.Items) == 0 {
		return false, nil
	}

	for i := range podList.Items {
		pod := podList.Items[i]
		// Check if the readiness conditions have been met
		conditions := pod.Status.Conditions
		if len(conditions) == 0 {
			return false, fmt.Errorf("Failed checking MySQL readiness gates, no status conditions found for pod %s/%s", pod.Namespace, pod.Name)
		}
		if !isPodReadinessGatesReady(pod, conditions) {
			return true, nil
		}
	}
	return false, nil
}

// isPodReadinessGatesReady - return boolean indicating if all readiness gate
// conditions have been met for the pod
func isPodReadinessGatesReady(pod v1.Pod, conditions []v1.PodCondition) bool {
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
	return len(pod.Spec.ReadinessGates) == readyCount
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
	nsn := types.NamespacedName{Namespace: componentNamespace, Name: helmReleaseName}
	if err := ctx.Client().Get(context.Background(), nsn, &innoDBCluster); err != nil {
		return nil, err
	}
	return &innoDBCluster, nil
}

// RepairMySQLPodStuckDeleting - temporary workaround to repair issue where a MySQL pod
// can be stuck terminating (e.g. during uninstall).  The workaround is to recycle the mysql-operator.
func (mc *MySQLChecker) RepairMySQLPodStuckDeleting() error {
	// Check if any MySQL pods are in the process of terminating
	selector := metav1.LabelSelectorRequirement{Key: mySQLComponentLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{mySQLDComponentName}}
	podList := k8sready.GetPodsList(mc.log, mc.client, types.NamespacedName{Namespace: componentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selector}})
	if podList == nil || len(podList.Items) == 0 {
		// No MySQL pods found, assume they have finished deleting
		resetInitialTimeMySQLPodsStuckChecked()
		return nil
	}

	foundPodsDeleting := false
	for i := range podList.Items {
		pod := podList.Items[i]
		if !pod.GetDeletionTimestamp().IsZero() {
			foundPodsDeleting = true
			break
		}
	}

	if foundPodsDeleting {
		// First time through start a timer
		if getInitialTimeMySQLPodsStuckChecked().IsZero() {
			setInitialTimeMySQLPodsStuckChecked(time.Now())
			mc.log.Progressf("Waiting for MySQL pods to terminate in namespace %s", componentNamespace)
			return nil
		}

		// Initiate repair only if time to wait period has been exceeded
		expiredTime := getInitialTimeMySQLPodsStuckChecked().Add(mc.RepairTimeout)
		if time.Now().After(expiredTime) {
			if err := restartMySQLOperator(mc.log, mc.client, "MySQL pods stuck terminating"); err != nil {
				return err
			}
		} else {
			// Keep trying until no pods deleting or timer expires
			return nil
		}
	}

	// Clear the timer
	resetInitialTimeMySQLPodsStuckChecked()
	return nil
}

// restartMySQLOperator - restart the MySQL Operator pod
func restartMySQLOperator(log vzlog.VerrazzanoLogger, client clipkg.Client, reason string) error {
	log.Info("Restarting the mysql-operator to see if it will repair: %s", reason)

	operPod, err := getMySQLOperatorPod(log, client)
	if err != nil {
		return fmt.Errorf("Failed restarting the mysql-operator to repair MySQL pods stuck deleting: %v", err)
	}

	if err = client.Delete(context.TODO(), operPod, &clipkg.DeleteOptions{}); err != nil {
		return err
	}

	// Reset all timers that have workarounds of restarting the MySQL operator.
	resetInitialTimeMySQLPodsStuckChecked()
	resetInitialTimeReadinessGateChecked()
	resetInitialTimeICUninstallChecked()

	return nil
}
