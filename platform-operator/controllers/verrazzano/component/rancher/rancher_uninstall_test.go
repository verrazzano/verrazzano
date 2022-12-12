// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"testing"

	asserts "github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type fakeMonitor struct {
	result  bool
	err     error
	running bool
}

func (f *fakeMonitor) run(args postUninstallRoutineParams) {
}

func (f *fakeMonitor) checkResult() (bool, error) { return f.result, f.err }

func (f *fakeMonitor) reset() {}

func (f *fakeMonitor) init() {}

func (f *fakeMonitor) sendResult(r bool) {}

func (f *fakeMonitor) isRunning() bool { return f.running }

var _ postUninstallMonitor = &fakeMonitor{}

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the objects exist in the cluster
// THEN the post-uninstall starts a new attempt and returns a RetryableError to requeue
func TestPostUninstall(t *testing.T) {
	// TODO: write this
}

// TestBackgroundPostUninstallCompletedSuccessfully tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the monitor goroutine fails to successfully complete
// THEN the post-uninstall returns nil without calling the forkPostUninstall function
func TestBackgroundPostUninstallCompletedSuccessfully(t *testing.T) {
	// TODO: write this
}

// TestPostUninstall tests the post uninstall process for Rancher
// GIVEN a call to postUninstall
// WHEN the the monitor goroutine failed to successfully complete
// THEN the postUninstall function calls the forkPostUninstall function and returns a retry error
func TestBackgroundPostUninstallRetryOnFailure(t *testing.T) {
	// TODO: write this
}

func Test_forkPostUninstallSuccess(t *testing.T) {
	// TODO: write this
}

func Test_forkPostUninstallFailure(t *testing.T) {
	// TODO: write this
}

// TestIsRancherNamespace tests the namespace belongs to Rancher
// GIVEN a call to isRancherNamespace
// WHEN the namespace belings to Rancher or not
// THEN we see true if it is and false if not
func TestIsRancherNamespace(t *testing.T) {
	// FIXME: perhaps need to change this?
	assert := asserts.New(t)

	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cattle-system",
		},
	}))
	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "true",
			},
		},
	}))
	assert.True(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	assert.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
			Annotations: map[string]string{
				rancherSysNS: "false",
			},
		},
	}))
	assert.False(isRancherNamespace(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "p-12345",
		},
	}))
}
