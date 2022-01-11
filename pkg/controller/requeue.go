// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package controller

import (
	"k8s.io/apimachinery/pkg/util/rand"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// Create a new Result that will cause a reconcile requeue after a short delay
func NewRequeueWithDelay(min int, max int, units time.Duration) ctrl.Result {
	return ctrl.Result{Requeue: true, RequeueAfter: CalculateDelay(min, max, units)}
}

//CalculateDelay - calculate a pseudo-random delay between min and max in the specified units
func CalculateDelay(min int, max int, units time.Duration) time.Duration {
	var seconds = rand.IntnRange(min, max)
	delaySecs := time.Duration(seconds) * units
	return delaySecs
}
