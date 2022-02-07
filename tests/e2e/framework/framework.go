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
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"math/rand"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

const (
	// DefaultNamespaceDeletionTimeout is timeout duration for waiting for a namespace deletion.
	DefaultNamespaceDeletionTimeout = 5 * time.Minute
)

// TODO Refactor time structs
// Time related declarations have been taken from a yet to exist e2e/framework/timeouts.go
// Storing in this file for simplicity for the time being
const (
	// Default timeouts to be used in TimeoutContext
	podStartTimeout                  = 5 * time.Minute
	podStartShortTimeout             = 2 * time.Minute
	podStartSlowTimeout              = 15 * time.Minute
	podDeleteTimeout                 = 5 * time.Minute
	claimProvisionTimeout            = 5 * time.Minute
	claimProvisionShortTimeout       = 1 * time.Minute
	dataSourceProvisionTimeout       = 5 * time.Minute
	claimBoundTimeout                = 3 * time.Minute
	pvReclaimTimeout                 = 3 * time.Minute
	pvBoundTimeout                   = 3 * time.Minute
	pvCreateTimeout                  = 3 * time.Minute
	pvDeleteTimeout                  = 3 * time.Minute
	pvDeleteSlowTimeout              = 20 * time.Minute
	snapshotCreateTimeout            = 5 * time.Minute
	snapshotDeleteTimeout            = 5 * time.Minute
	snapshotControllerMetricsTimeout = 5 * time.Minute
)

// TimeoutContext contains timeout settings for several actions.
type TimeoutContext struct {
	// PodStart is how long to wait for the pod to be started.
	PodStart time.Duration

	// PodStartShort is same as `PodStart`, but shorter.
	// Use it in a case-by-case basis, mostly when you are sure pod start will not be delayed.
	PodStartShort time.Duration

	// PodStartSlow is same as `PodStart`, but longer.
	// Use it in a case-by-case basis, mostly when you are sure pod start will take longer than usual.
	PodStartSlow time.Duration

	// PodDelete is how long to wait for the pod to be deleted.
	PodDelete time.Duration

	// ClaimProvision is how long claims have to become dynamically provisioned.
	ClaimProvision time.Duration

	// DataSourceProvision is how long claims have to become dynamically provisioned from source claim.
	DataSourceProvision time.Duration

	// ClaimProvisionShort is the same as `ClaimProvision`, but shorter.
	ClaimProvisionShort time.Duration

	// ClaimBound is how long claims have to become bound.
	ClaimBound time.Duration

	// PVReclaim is how long PVs have to become reclaimed.
	PVReclaim time.Duration

	// PVBound is how long PVs have to become bound.
	PVBound time.Duration

	// PVCreate is how long PVs have to be created.
	PVCreate time.Duration

	// PVDelete is how long PVs have to become deleted.
	PVDelete time.Duration

	// PVDeleteSlow is the same as PVDelete, but slower.
	PVDeleteSlow time.Duration

	// SnapshotCreate is how long for snapshot to create snapshotContent.
	SnapshotCreate time.Duration

	// SnapshotDelete is how long for snapshot to delete snapshotContent.
	SnapshotDelete time.Duration

	// SnapshotControllerMetrics is how long to wait for snapshot controller metrics.
	SnapshotControllerMetrics time.Duration
}

// NewTimeoutContextWithDefaults returns a TimeoutContext with default values.
func NewTimeoutContextWithDefaults() *TimeoutContext {
	return &TimeoutContext{
		PodStart:                  podStartTimeout,
		PodStartShort:             podStartShortTimeout,
		PodStartSlow:              podStartSlowTimeout,
		PodDelete:                 podDeleteTimeout,
		ClaimProvision:            claimProvisionTimeout,
		ClaimProvisionShort:       claimProvisionShortTimeout,
		DataSourceProvision:       dataSourceProvisionTimeout,
		ClaimBound:                claimBoundTimeout,
		PVReclaim:                 pvReclaimTimeout,
		PVBound:                   pvBoundTimeout,
		PVCreate:                  pvCreateTimeout,
		PVDelete:                  pvDeleteTimeout,
		PVDeleteSlow:              pvDeleteSlowTimeout,
		SnapshotCreate:            snapshotCreateTimeout,
		SnapshotDelete:            snapshotDeleteTimeout,
		SnapshotControllerMetrics: snapshotControllerMetricsTimeout,
	}
}

// Framework supports common operations used by e2e tests; it will keep a client & a namespace for you.
// Eventual goal is to merge this with integration test framework.
type Framework struct {
	BaseName string

	// Set together with creating the ClientSet and the namespace.
	// Guaranteed to be unique in the cluster even when running the same
	// test multiple times in parallel.
	UniqueName string

	clientConfig                     *rest.Config

	DynamicClient dynamic.Interface

	SkipNamespaceCreation    bool            // Whether to skip creating a namespace
	Namespace                *v1.Namespace   // Every test has at least one namespace unless creation is skipped
	namespacesToDelete       []*v1.Namespace // Some tests have more than one.
	NamespaceDeletionTimeout time.Duration
	SkipPrivilegedPSPBinding bool // Whether to skip creating a binding to the privileged PSP in the test namespace

	// afterEaches is a map of name to function to be called after each test.  These are not
	// cleared.  The call order is randomized so that no dependencies can grow between
	// the various afterEaches
	afterEaches map[string]AfterEachActionFunc

	// beforeEachStarted indicates that BeforeEach has started
	beforeEachStarted bool

	// configuration for framework's client
	Options Options

	// Place where various additional data is stored during test run to be printed to ReportDir,
	// or stdout if ReportDir is not set once test ends.
	TestSummaries []TestDataSummary

	// Timeouts contains the custom timeouts used during the test execution.
	Timeouts *TimeoutContext
}

// AfterEachActionFunc is a function that can be called after each test
type AfterEachActionFunc func(f *Framework, failed bool)

// TestDataSummary is an interface for managing test data.
type TestDataSummary interface {
	SummaryKind() string
	PrintHumanReadable() string
	PrintJSON() string
}

// Options is a struct for managing test framework options.
type Options struct {
	ClientQPS    float32
	ClientBurst  int
	GroupVersion *schema.GroupVersion
}