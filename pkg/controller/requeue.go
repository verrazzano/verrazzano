// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package controller

import (
	"time"

	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Create a new Result that will cause a reconcile requeue after a delay within the specified range
func NewRequeueWithDelay(min int, max int, units time.Duration) ctrl.Result {
	var seconds = rand.IntnRange(min, max)
	delaySecs := time.Duration(seconds) * units
	return ctrl.Result{Requeue: true, RequeueAfter: delaySecs}
}

// ShortRequeue returns a new Result that will cause a reconcile requeue after a short delay
func ShortRequeue() ctrl.Result {
	return NewRequeueWithDelay(2, 3, time.Second)
}

// LongRequeue returns a new Result that will cause a reconcile requeue after a long delay (2 -3 minutes)
func LongRequeue() ctrl.Result {
	return NewRequeueWithDelay(2, 3, time.Minute)
}

// Return true if requeue is needed
func ShouldRequeue(r ctrl.Result) bool {
	return r.Requeue || r.RequeueAfter > 0
}
