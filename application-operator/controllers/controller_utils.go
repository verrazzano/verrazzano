// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"strconv"
	"strings"
)

// ConvertAPIVersionToGroupAndVersion splits APIVersion into API and version parts.
// An APIVersion takes the form api/version (e.g. networking.k8s.io/v1)
// If the input does not contain a / the group is defaulted to the empty string.
// apiVersion - The combined api and version to split
func ConvertAPIVersionToGroupAndVersion(apiVersion string) (string, string) {
	parts := strings.SplitN(apiVersion, "/", 2)
	if len(parts) < 2 {
		// Use empty group for core types.
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// EnsureLastGenerationInStatus ensures that the status has the last generation saved
func EnsureLastGenerationInStatus(client clipkg.Client, wl *vzapi.VerrazzanoWebLogicWorkload) (ctrl.Result, error) {
	if len(wl.Status.LastGeneration) > 0 {
		return ctrl.Result{}, nil
	}

	// Update the status generation and always requeue
	wl.Status.LastGeneration = strconv.Itoa(int(wl.Generation))
	err := client.Status().Update(context.TODO(), wl)
	return ctrl.Result{Requeue: true, RequeueAfter: 1}, err
}

// NewDefaultRateLimiter returns a RateLimiter with default base backoff and max backoff
func NewDefaultRateLimiter() workqueue.RateLimiter {
	// Default base delay for controller runtime requeue
	const BaseDelay = 5

	// Default maximum delay for controller runtime requeue
	const MaxDelay = 60

	return workqueue.NewItemExponentialFailureRateLimiter(
		secsToDuration(BaseDelay),
		secsToDuration(MaxDelay))
}

func secsToDuration(secs int) time.Duration {
	return time.Duration(float64(secs) * float64(time.Second))
}
