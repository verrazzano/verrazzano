// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

// Reusable code for Quick Create controllersΩ©

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

func RequeueDelay() ctrl.Result {
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: 30 * time.Second,
	}
}
