// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

/*
Inspired by The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"reflect"
	"runtime"
	"sync"
)

// CleanupActionHandle is an integer pointer type for handling cleanup action
type CleanupActionHandle *int
type cleanupFuncHandle struct {
	actionHandle CleanupActionHandle
	actionHook   func()
}

var cleanupActionsLock sync.Mutex
var cleanupHookList = []cleanupFuncHandle{}

// AddCleanupAction installs a function that will be called in the event of the
// whole test being terminated.  This allows arbitrary pieces of the overall
// test to hook into SynchronizedAfterSuite().
// The hooks are called in last-in-first-out order.
func AddCleanupAction(fn func()) CleanupActionHandle {
	p := CleanupActionHandle(new(int))
	cleanupActionsLock.Lock()
	defer cleanupActionsLock.Unlock()
	c := cleanupFuncHandle{actionHandle: p, actionHook: fn}
	cleanupHookList = append([]cleanupFuncHandle{c}, cleanupHookList...)
	return p
}

// RemoveCleanupAction removes a function that was installed by
// AddCleanupAction.
func RemoveCleanupAction(p CleanupActionHandle) {
	cleanupActionsLock.Lock()
	defer cleanupActionsLock.Unlock()
	for i, item := range cleanupHookList {
		if item.actionHandle == p {
			cleanupHookList = append(cleanupHookList[:i], cleanupHookList[i+1:]...)
			break
		}
	}
}

// RunCleanupActions runs all functions installed by AddCleanupAction.  It does
// not remove them (see RemoveCleanupAction) but it does run unlocked, so they
// may remove themselves.
func RunCleanupActions() {
	list := []func(){}
	func() {
		cleanupActionsLock.Lock()
		defer cleanupActionsLock.Unlock()
		for _, p := range cleanupHookList {
			list = append(list, p.actionHook)
		}
	}()
	// Run unlocked.
	for _, fn := range list {
		Logf("Running Cleanup Action: %v", runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name())
		fn()
	}
}
