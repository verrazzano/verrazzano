// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	mySQLComponentLabel      = "component"
	mySQLDComponentName      = "mysqld"
	helmReleaseName          = "mysql"
	componentNamespace       = "keycloak"
	componentName            = "mysql"
	mysqlRouterComponentName = "mysqlrouter"

	// Alert Names
	alertPodStuckTerminating = "pod-stuck"
	alertInnoDBCluster       = "innodbcluster"
	alertMySQLOperator       = "mysql-operator"
	alertReadinessGate       = "readiness-gate"
	alertMySQLRouter         = "mysql-router"
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
		ctx.Log().Debugf("Starting check to insure the InnoDBCluster %s/%s is not stuck deleting", componentNamespace, helmReleaseName)
		return nil
	}

	// Initiate repair only if time to wait period has been exceeded
	expiredTime := getInitialTimeICUninstallChecked().Add(GetMySQLChecker().RepairTimeout)
	if time.Now().After(expiredTime) {
		msg := fmt.Sprintf("InnoDBCluster stuck deleting for a minimum of %s", GetMySQLChecker().RepairTimeout.String())
		createEvent(ctx.Log(), ctx.Client(), innoDBCluster, alertInnoDBCluster, "StuckDeleting", msg)
		return restartMySQLOperator(ctx.Log(), ctx.Client(), msg)
	}
	return nil
}

// RepairMySQLPodsWaitingReadinessGates - temporary workaround to repair issue were a MySQL pod
// can be stuck waiting for its readiness gates to be met.  The workaround is to recycle the mysql-operator.
func (mc *MySQLChecker) RepairMySQLPodsWaitingReadinessGates() error {
	podsWaiting, err := getPodsWaitingForReadinessGates(mc.log, mc.client)
	if err != nil {
		return err
	}
	if len(podsWaiting) > 0 {
		// Start a timer the first time pods are waiting for readiness gates
		if getLastTimeReadinessGateChecked().IsZero() {
			setInitialTimeReadinessGateChecked(time.Now())
			return nil
		}

		// Initiate repair only if time to wait period has been exceeded
		expiredTime := getLastTimeReadinessGateChecked().Add(mc.RepairTimeout)
		if time.Now().After(expiredTime) {
			for i := range podsWaiting {
				mc.logEvent(podsWaiting[i], alertReadinessGate, "WaitingReadinessGate", fmt.Sprintf("Pod stuck waiting for readiness gates for a minimum of %s", mc.RepairTimeout.String()))
			}
			return restartMySQLOperator(mc.log, mc.client, "MySQL pods waiting for readiness gates")
		}
	}

	// Clear the timer when no pods are waiting
	resetInitialTimeReadinessGateChecked()
	return nil
}

// getPodsWaitingForReadinessGates - return the list of pods waiting for readiness gates
func getPodsWaitingForReadinessGates(log vzlog.VerrazzanoLogger, client clipkg.Client) ([]v1.Pod, error) {
	var podsWaiting []v1.Pod

	log.Debug("Checking if MySQL pods waiting for readiness gates")

	selector := metav1.LabelSelectorRequirement{Key: mySQLComponentLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{mySQLDComponentName}}
	podList := k8sready.GetPodsList(log, client, types.NamespacedName{Namespace: componentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selector}})
	if podList == nil || len(podList.Items) == 0 {
		return podsWaiting, nil
	}

	for i := range podList.Items {
		pod := podList.Items[i]
		// Check if the readiness conditions have been met
		conditions := pod.Status.Conditions
		if len(conditions) == 0 {
			return podsWaiting, fmt.Errorf("Failed checking MySQL readiness gates, no status conditions found for pod %s/%s", pod.Namespace, pod.Name)
		}
		if !isPodReadinessGatesReady(pod, conditions) {
			podsWaiting = append(podsWaiting, pod)
		}
	}
	return podsWaiting, nil
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

// RepairMySQLPodStuckTerminating - temporary workaround to repair issue where a MySQL pod
// can be stuck terminating (e.g. during uninstall).  The workaround is to recycle the mysql-operator.
func (mc *MySQLChecker) RepairMySQLPodStuckTerminating() error {
	// Check if any MySQL pods are in the process of terminating
	selector := metav1.LabelSelectorRequirement{Key: mySQLComponentLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{mySQLDComponentName}}
	podList := k8sready.GetPodsList(mc.log, mc.client, types.NamespacedName{Namespace: componentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selector}})
	if podList == nil || len(podList.Items) == 0 {
		// No MySQL pods found, assume they have finished deleting
		resetInitialTimeMySQLPodsStuckChecked()
		return nil
	}

	foundPodsDeleting := false
	podStuckDeleting := v1.Pod{}
	for i := range podList.Items {
		pod := podList.Items[i]
		if !pod.GetDeletionTimestamp().IsZero() {
			foundPodsDeleting = true
			podStuckDeleting = pod
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
			mc.logEvent(podStuckDeleting, alertPodStuckTerminating, "PodStuckTerminating", fmt.Sprintf("Pod stuck deleting for a minimum of %s", mc.RepairTimeout.String()))
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

// RepairMySQLRouterPodsCrashLoopBackoff - repair mysql-router pods stuck in CrashLoopBackoff.
// The workaround is to delete the pod.
func (mc *MySQLChecker) RepairMySQLRouterPodsCrashLoopBackoff() error {
	selector := metav1.LabelSelectorRequirement{Key: mySQLComponentLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{mysqlRouterComponentName}}
	podList := k8sready.GetPodsList(mc.log, mc.client, types.NamespacedName{Namespace: componentNamespace}, &metav1.LabelSelector{MatchExpressions: []metav1.LabelSelectorRequirement{selector}})
	if podList == nil || len(podList.Items) == 0 {
		// No MySQL pods found
		return nil
	}

	for i := range podList.Items {
		pod := podList.Items[i]
		for _, container := range pod.Status.ContainerStatuses {
			if waiting := container.State.Waiting; waiting != nil {
				if waiting.Reason == "CrashLoopBackOff" {
					// Terminate the pod
					msg := fmt.Sprintf("Terminating pod %s/%s because it is stuck in CrashLoopBackOff", pod.Namespace, pod.Name)
					mc.log.Infof("%s", msg)
					mc.logEvent(pod, alertMySQLRouter, waiting.Reason, msg)
					if err := mc.client.Delete(context.TODO(), &pod, &clipkg.DeleteOptions{}); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// restartMySQLOperator - restart the MySQL Operator pod
func restartMySQLOperator(log vzlog.VerrazzanoLogger, client clipkg.Client, reason string) error {
	operPod, err := getMySQLOperatorPod(log, client)
	if err != nil {
		msg := fmt.Sprintf("Failed restarting the mysql-operator to repair stuck resources: %v", err)
		createEvent(log, client, metav1.ObjectMeta{Namespace: mysqloperator.ComponentNamespace}, alertMySQLOperator, "PodNotFound", msg)
		return fmt.Errorf("%s", msg)
	}
	message := fmt.Sprintf("Restarting the mysql-operator to repair: %s", reason)
	log.Infof(message)
	createEvent(log, client, operPod, alertMySQLOperator, "RestartMySQLOperator", message)

	if err = client.Delete(context.TODO(), operPod, &clipkg.DeleteOptions{}); err != nil {
		createEvent(log, client, operPod, alertMySQLOperator, "PodNotDeleted", fmt.Sprintf("Failed to delete the mysql-operator pod: %v", err))
		return err
	}

	// Reset all timers that have workarounds of restarting the MySQL operator.
	resetInitialTimeMySQLPodsStuckChecked()
	resetInitialTimeReadinessGateChecked()
	resetInitialTimeICUninstallChecked()

	return nil
}

func (mc *MySQLChecker) logEvent(involvedObject interface{}, alertName string, reason string, message string) {
	createEvent(mc.log, mc.client, involvedObject, alertName, reason, message)
}

// createEvent - create or update an event
func createEvent(log vzlog.VerrazzanoLogger, client clipkg.Client, objectInt interface{}, alertName string, reason string, message string) {
	event := &v1.Event{}
	ctx := context.TODO()

	// Convert involved object to unstructured
	unstructuredInt, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&objectInt)
	if err != nil {
		log.ErrorfThrottled("%v", err)
		return
	}
	involvedObject := unstructured.Unstructured{Object: unstructuredInt}

	err = client.Get(ctx, types.NamespacedName{Namespace: involvedObject.GetNamespace(), Name: generateAlertName(alertName)}, event)
	if err != nil && !errors.IsNotFound(err) {
		log.ErrorfThrottled("%v", err)
	}

	// Create a new event if not found
	if err != nil {
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateAlertName(alertName),
				Namespace: involvedObject.GetNamespace(),
			},
			InvolvedObject: func() v1.ObjectReference {
				objectRef := v1.ObjectReference{}
				setObjectReference(&objectRef, involvedObject)
				return objectRef
			}(),
			Type:                "Warning",
			FirstTimestamp:      metav1.Now(),
			LastTimestamp:       metav1.Now(),
			Reason:              reason,
			Message:             message,
			ReportingController: controllerName,
		}

		if err := client.Create(context.TODO(), event); err != nil {
			log.ErrorfThrottled("%v", err)
		}
	} else {
		// Update the existing event
		event.Reason = reason
		event.Message = message
		event.LastTimestamp = metav1.Now()
		setObjectReference(&event.InvolvedObject, involvedObject)
		if err := client.Update(context.TODO(), event); err != nil {
			log.ErrorfThrottled("%v", err)
		}
	}
}

// setObjectReference - populate the ObjectReference object from the involved object
func setObjectReference(objectRef *v1.ObjectReference, involvedObject unstructured.Unstructured) {
	objectRef.Kind = involvedObject.GetKind()
	objectRef.APIVersion = involvedObject.GetAPIVersion()
	objectRef.Namespace = involvedObject.GetNamespace()
	objectRef.Name = involvedObject.GetName()
	objectRef.UID = involvedObject.GetUID()
	objectRef.ResourceVersion = involvedObject.GetResourceVersion()
}

// generateAlertName - generate the alert name
func generateAlertName(alertName string) string {
	return fmt.Sprintf("verrazzano-%s", alertName)
}
