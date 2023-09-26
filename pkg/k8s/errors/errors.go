// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package errors

import (
	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
)

func IsUnauthorized(err error) bool {
	if apierrors.IsUnauthorized(err) {
		return true
	}
	groupErr := &discovery.ErrGroupDiscoveryFailed{}
	if errors.As(err, &groupErr) {
		for _, err := range groupErr.Groups {
			if apierrors.IsUnauthorized(err) {
				return true
			}
		}
	}
	return false
}

func IsNotFound(err error) bool {
	if apierrors.IsNotFound(err) {
		return true
	}
	groupErr := &discovery.ErrGroupDiscoveryFailed{}
	if errors.As(err, &groupErr) {
		for _, err := range groupErr.Groups {
			if apierrors.IsNotFound(err) {
				return true
			}
		}
	}
	return false
}

func IsForbidden(err error) bool {
	if apierrors.IsForbidden(err) {
		return true
	}
	groupErr := &discovery.ErrGroupDiscoveryFailed{}
	if errors.As(err, &groupErr) {
		for _, err := range groupErr.Groups {
			if apierrors.IsForbidden(err) {
				return true
			}
		}
	}
	return false
}

func IsConflict(err error) bool {
	if apierrors.IsConflict(err) {
		return true
	}
	groupErr := &discovery.ErrGroupDiscoveryFailed{}
	if errors.As(err, &groupErr) {
		for _, err := range groupErr.Groups {
			if apierrors.IsConflict(err) {
				return true
			}
		}
	}
	return false
}

func IsAlreadyExists(err error) bool {
	if apierrors.IsAlreadyExists(err) {
		return true
	}
	groupErr := &discovery.ErrGroupDiscoveryFailed{}
	if errors.As(err, &groupErr) {
		for _, err := range groupErr.Groups {
			if apierrors.IsAlreadyExists(err) {
				return true
			}
		}
	}
	return false
}

func NewNotFound(qualifiedResource schema.GroupResource, name string) *apierrors.StatusError {
	return apierrors.NewNotFound(qualifiedResource, name)
}

func NewBadRequest(reason string) *apierrors.StatusError {
	return apierrors.NewBadRequest(reason)
}

func NewTooManyRequests(message string, retryAfterSeconds int) *apierrors.StatusError {
	return apierrors.NewTooManyRequests(message, retryAfterSeconds)
}

func NewApplyConflict(causes []metav1.StatusCause, message string) *apierrors.StatusError {
	return apierrors.NewApplyConflict(causes, message)
}

func NewResourceExpired(message string) *apierrors.StatusError {
	return apierrors.NewResourceExpired(message)
}

func NewInternalError(err error) *apierrors.StatusError {
	return apierrors.NewInternalError(err)
}
