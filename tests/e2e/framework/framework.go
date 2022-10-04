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
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"reflect"
	"strings"
)

// Default labels for the namespace
var nsLabels = map[string]string{
	"verrazzano-managed": "true",
	"istio-injection":    "enabled",
}

// Framework supports common operations used by e2e tests; it will keep a client & a namespace for you.
// Eventual goal is to merge this with integration test framework.
type Framework struct {
	BaseName string

	// Set together with creating the ClientSet and the namespace.
	// Guaranteed to be unique in the cluster even when running the same
	// test multiple times in parallel.
	UniqueName string

	clientConfig *rest.Config
	ClientSet    *clientset.Clientset

	DynamicClient dynamic.Interface

	SkipNamespaceCreation bool            // Whether to skip creating a namespace
	Namespace             *v1.Namespace   // Every test has at least one namespace unless creation is skipped
	namespacesToDelete    []*v1.Namespace // Some tests have more than one.

	NamespaceAnnotations map[string]string

	KubeConfig string // Custom kube config

	// afterEaches is a map of name to function to be called after each test.  These are not
	// cleared.  The call order is randomized so that no dependencies can grow between
	// the various afterEaches
	afterEaches map[string]AfterEachActionFunc

	// beforeEachStarted indicates that BeforeEach has started
	beforeEachStarted bool

	// fields associated with metrics + logging
	Metrics *zap.SugaredLogger
	Logs    *zap.SugaredLogger
}

// AfterEachActionFunc is a function that can be called after each test
type AfterEachActionFunc func(f *Framework, failed bool)

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	return NewFramework(baseName, "", nil)
}

// NewDefaultFrameworkWithKubeConfig is similar NewDefaultFramework, with additional support for custom kube configuration
func NewDefaultFrameworkWithKubeConfig(baseName string, kubeConfig string) *Framework {
	return NewFramework(baseName, kubeConfig, nil)
}

// NewFramework creates a test framework.
func NewFramework(baseName string, kubeconfig string, client *clientset.Clientset) *Framework {
	metricsIndex, _ := metrics.NewLogger(baseName, metrics.MetricsIndex)
	logIndex, _ := metrics.NewLogger(baseName, metrics.TestLogIndex, "stdout")

	f := &Framework{
		BaseName:   strings.ToLower(baseName),
		ClientSet:  client,
		Metrics:    metricsIndex,
		Logs:       logIndex,
		KubeConfig: kubeconfig,
	}
	f.UniqueName = pkg.GenerateNamespace(strings.ToLower(baseName))

	f.AddAfterEach("dumpNamespaceInfo", func(f *Framework, failed bool) {
		if !failed {
			return
		}
	})

	ginkgo.BeforeEach(f.BeforeEach)
	ginkgo.AfterEach(f.AfterEach)

	return f
}

// BeforeEach gets a client and makes a namespace.
func (f *Framework) BeforeEach() {
	f.beforeEachStarted = true

	if f.ClientSet == nil {
		ginkgo.By("Creating a kubernetes client")

		var config *rest.Config
		var err error
		if f.KubeConfig != "" {
			config, err = k8sutil.GetKubeConfigGivenPath(f.KubeConfig)
		} else {
			config, err = k8sutil.GetKubeConfig()
		}
		ExpectNoError(err)

		f.clientConfig = rest.CopyConfig(config)

		f.ClientSet, err = k8sutil.GetKubernetesClientsetWithConfig(config)
		ExpectNoError(err)

		f.DynamicClient, err = dynamic.NewForConfig(config)
		ExpectNoError(err)

		// Create namespace for the tests
		if !f.SkipNamespaceCreation {
			ns, _ := f.CreateNamespace(f.UniqueName, nsLabels, f.ClientSet, f.NamespaceAnnotations)
			f.AddNamespacesToDelete(ns)
		}
	}
}

// AddAfterEach is a way to add a function to be called after every test. The execution order is intentionally random
// to avoid growing dependencies.  If you register the same name twice, it is a coding error and will panic.
func (f *Framework) AddAfterEach(name string, fn AfterEachActionFunc) {
	if _, ok := f.afterEaches[name]; ok {
		panic(fmt.Sprintf("%q is already registered", name))
	}

	if f.afterEaches == nil {
		f.afterEaches = map[string]AfterEachActionFunc{}
	}
	f.afterEaches[name] = fn
}

// AfterEach deletes the namespace, after reading its events.
func (f *Framework) AfterEach() {
	// If BeforeEach never started AfterEach should be skipped.
	if !f.beforeEachStarted {
		return
	}

	// ClientSet should not be nil, indicates a bad state for the remaining tests
	if f.ClientSet == nil {
		panic("The ClientSet defined by the framework must not be nil in AfterEach")
	}
}

// ClientConfig an externally accessible method for reading the kube client config.
func (f *Framework) ClientConfig() *rest.Config {
	ret := rest.CopyConfig(f.clientConfig)
	// json is least common denominator
	ret.ContentType = runtime.ContentTypeJSON
	ret.AcceptContentTypes = runtime.ContentTypeJSON
	return ret
}

// CreateNamespace creates the namespace specified by the name
func (f *Framework) CreateNamespace(name string, labels map[string]string, client *clientset.Clientset, annotations map[string]string) (*v1.Namespace, error) {
	return pkg.CreateNamespaceWithClientSet(name, labels, client, annotations)
}

// AddNamespacesToDelete adds one or more namespaces to be deleted when the test
// completes.
func (f *Framework) AddNamespacesToDelete(namespaces ...*v1.Namespace) {
	for _, ns := range namespaces {
		if ns == nil {
			continue
		}
		f.namespacesToDelete = append(f.namespacesToDelete, ns)
	}
}

// DeleteNamespace deletes the namespace specified by the name
func (f *Framework) DeleteNamespace(name string) {
	if len(os.Getenv(k8sutil.EnvVarTestKubeConfig)) > 0 {
		pkg.Log(pkg.Info, fmt.Sprintf("DeleteNamespace %s, test is running with custom service account and "+
			"therefore namespace won't be deleted by the test", name))
		return
	}
	pkg.DeleteNamespaceWithClientSet(name, f.ClientSet)
}

// CheckNSFinalizersRemoved checks whether the namespace finalizers are removed
func (f *Framework) CheckNSFinalizersRemoved(name string) bool {
	return pkg.CheckNSFinalizerRemoved(name, f.ClientSet)
}

// Ginkgo wrapper functions start from here
// It wraps Ginkgo It to emit a metric
func (f *Framework) It(text string, args ...interface{}) bool {
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}
	body := args[len(args)-1]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	fn := func() {
		metrics.Emit(f.Metrics.With(metrics.Status, metrics.Started))
		reflect.ValueOf(body).Call([]reflect.Value{})
	}

	args[len(args)-1] = ginkgo.Offset(1)
	args = append(args, fn)
	return ginkgo.It(text, args...)
}

// Describe wraps Ginkgo Describe to emit a metric
func (f *Framework) Describe(text string, args ...interface{}) bool {
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}
	body := args[len(args)-1]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	fn := func() {
		metrics.Emit(f.Metrics.With(metrics.Status, metrics.Started))
		reflect.ValueOf(body).Call([]reflect.Value{})
		metrics.Emit(f.Metrics.With(metrics.Duration, metrics.DurationMillis()))
	}
	args[len(args)-1] = ginkgo.Offset(1)
	args = append(args, fn)
	return ginkgo.Describe(text, args...)
}

// By wraps Ginkgo By to emit a metric
func (f *Framework) By(text string, args ...func()) {
	if len(args) > 1 {
		panic("More than just one callback per By, please")
	}
	metrics.Emit(f.Metrics.With(metrics.Status, metrics.Started))
	if len(args) == 1 {
		if !isBodyFunc(args[0]) {
			ginkgo.Fail("Unsupported body type - expected function")
		}
		fn := func() {
			reflect.ValueOf(args[0]).Call([]reflect.Value{})
		}
		ginkgo.By(text, fn)
	} else {
		ginkgo.By(text)
	}
}

// DescribeTable - wrapper function for Ginkgo DescribeTable
func (f *Framework) DescribeTable(text string, args ...interface{}) bool {
	if args == nil {
		ginkgo.Fail("Unsupported args type - expected non-nil")
	}
	body := args[0]
	if !isBodyFunc(body) {
		ginkgo.Fail("Unsupported body type - expected function")
	}
	funcType := reflect.TypeOf(body)
	fn := reflect.MakeFunc(funcType, func(args []reflect.Value) (results []reflect.Value) {
		metrics.Emit(f.Metrics.With(metrics.Status, metrics.Started))
		rv := reflect.ValueOf(body).Call(args)
		metrics.Emit(f.Metrics.With(metrics.Duration, metrics.DurationMillis()))
		return rv
	})
	args[0] = fn.Interface()
	return ginkgo.DescribeTable(text, args...)
}

// Entry - wrapper function for Ginkgo Entry
func (f *Framework) Entry(description interface{}, args ...interface{}) ginkgo.TableEntry {
	// insert an Offset into the args, but not as the last item, so that the right code location is reported
	fn := args[len(args)-1]
	args[len(args)-1] = ginkgo.Offset(6) // need to go 6 up the stack to get the caller
	args = append(args, fn)
	return ginkgo.Entry(description, args...)
}

// Fail - wrapper function for Ginkgo Fail
func (f *Framework) Fail(message string, callerSkip ...int) {
	ginkgo.Fail(message, callerSkip...)
}

// Context - wrapper function for Ginkgo Context
func (f *Framework) Context(text string, args ...interface{}) bool {
	return f.Describe(text, args...)
}

// When - wrapper function for Ginkgo When
func (f *Framework) When(text string, args ...interface{}) bool {
	return ginkgo.When(text, args...)
}

// SynchronizedBeforeSuite - wrapper function for Ginkgo SynchronizedBeforeSuite
func (f *Framework) SynchronizedBeforeSuite(process1Body func() []byte, allProcessBody func([]byte)) bool {
	return ginkgo.SynchronizedBeforeSuite(process1Body, allProcessBody)
}

// SynchronizedAfterSuite - wrapper function for Ginkgo SynchronizedAfterSuite
func (f *Framework) SynchronizedAfterSuite(allProcessBody func(), process1Body func()) bool {
	return ginkgo.SynchronizedAfterSuite(allProcessBody, process1Body)
}

// JustBeforeEach - wrapper function for Ginkgo JustBeforeEach
func (f *Framework) JustBeforeEach(args ...interface{}) bool {
	return ginkgo.JustBeforeEach(args...)
}

// JustAfterEach - wrapper function for Ginkgo JustAfterEach
func (f *Framework) JustAfterEach(args ...interface{}) bool {
	return ginkgo.JustAfterEach(args...)
}

// BeforeAll - wrapper function for Ginkgo BeforeAll
func (f *Framework) BeforeAll(args ...interface{}) bool {
	return ginkgo.BeforeAll(args...)
}

// AfterAll - wrapper function for Ginkgo AfterAll
func (f *Framework) AfterAll(args ...interface{}) bool {
	return ginkgo.AfterAll(args...)
}

// isBodyFunc - return boolean indicating if the interface is a function
func isBodyFunc(body interface{}) bool {
	bodyType := reflect.TypeOf(body)
	return bodyType.Kind() == reflect.Func
}
