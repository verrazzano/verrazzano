// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqlcheck

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Event Names
	eventPodStuckTerminating = "pod-stuck"
	eventInnoDBCluster       = "innodbcluster"
	eventMySQLOperator       = "mysql-operator"
	eventReadinessGate       = "readiness-gate"
	eventMySQLRouter         = "mysql-router"

	// Event Reasons
	reasonStuckDeleting        = "StuckDeleting"
	reasonStuckTerminating     = "StuckTerminating"
	reasonWaitingReadinessGate = "WaitingReadinessGate"
	reasonNotFound             = "NotFound"
	reasonNotDeleted           = "NotDeleted"
	reasonCrashLoopBackOff     = "CrashLoopBackOff"
	reasonRestart              = "Restart"
)

func (mc *MySQLChecker) logEvent(involvedObject interface{}, eventName string, reason string, message string) {
	createEvent(mc.log, mc.client, involvedObject, eventName, reason, message)
}

// createEvent - create or update an event, and record the message in the log file
func createEvent(log vzlog.VerrazzanoLogger, client clipkg.Client, objectInt interface{}, eventName string, reason string, message string) {
	event := &v1.Event{}
	ctx := context.TODO()

	// Convert involved object to unstructured
	unstructuredInt, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&objectInt)
	if err != nil {
		log.ErrorfThrottled("%v", err)
		return
	}
	involvedObject := unstructured.Unstructured{Object: unstructuredInt}

	err = client.Get(ctx, types.NamespacedName{Namespace: involvedObject.GetNamespace(), Name: generateEventName(eventName)}, event)
	if err != nil && !errors.IsNotFound(err) {
		log.ErrorfThrottled("%v", err)
	}

	// Create a new event if not found
	if err != nil {
		event := &v1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      generateEventName(eventName),
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

	// Record the message in the log file in addition to generating an event
	log.Info(message)
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

// generateEventName - generate the event name
func generateEventName(eventName string) string {
	return fmt.Sprintf("verrazzano-%s", eventName)
}
