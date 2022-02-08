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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
)

//const (
//	// DefaultNamespaceDeletionTimeout is timeout duration for waiting for a namespace deletion.
//	DefaultNamespaceDeletionTimeout = 5 * time.Minute
//)

// Framework supports common operations used by e2e tests; it will keep a client & a namespace for you.
// Eventual goal is to merge this with integration test framework.
type Framework struct {
	BaseName string

	// Set together with creating the ClientSet and the namespace.
	// Guaranteed to be unique in the cluster even when running the same
	// test multiple times in parallel.
	UniqueName string

	clientConfig			*rest.Config
	ClientSet               clientset.Interface

	DynamicClient 			dynamic.Interface

	SkipNamespaceCreation    bool            // Whether to skip creating a namespace
	Namespace                *v1.Namespace   // Every test has at least one namespace unless creation is skipped
	namespacesToDelete       []*v1.Namespace // Some tests have more than one.

	// afterEaches is a map of name to function to be called after each test.  These are not
	// cleared.  The call order is randomized so that no dependencies can grow between
	// the various afterEaches
	afterEaches map[string]AfterEachActionFunc

	// beforeEachStarted indicates that BeforeEach has started
	beforeEachStarted bool
}

// AfterEachActionFunc is a function that can be called after each test
type AfterEachActionFunc func(f *Framework, failed bool)

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	return NewFramework(baseName, nil)
}

// NewFramework creates a test framework.
func NewFramework(baseName string, client clientset.Interface) *Framework {
	f := &Framework{
		BaseName:                 baseName,
		ClientSet:                client,
	}

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
		config, err := k8sutil.GetKubeConfig()
		ExpectNoError(err)

		f.clientConfig = rest.CopyConfig(config)

		f.ClientSet, err = k8sutil.GetKubernetesClientset()
		ExpectNoError(err)

		f.DynamicClient, err = dynamic.NewForConfig(config)
		ExpectNoError(err)
	}
}

// AddAfterEach is a way to add a function to be called after every test.  The execution order is intentionally random
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
	// Currently some tests under e2e/storage have this condition.
	if !f.beforeEachStarted {
		return
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

// ExpectNoError checks if "err" is set, and if so, fails assertion while logging the error.
func ExpectNoError(err error, explain ...interface{}) {
	ExpectNoErrorWithOffset(1, err, explain...)
}

// ExpectNoErrorWithOffset checks if "err" is set, and if so, fails assertion while logging the error at "offset" levels above its caller
// (for example, for call chain f -> g -> ExpectNoErrorWithOffset(1, ...) error would be logged for "f").
func ExpectNoErrorWithOffset(offset int, err error, explain ...interface{}) {
	gomega.ExpectWithOffset(1+offset, err).NotTo(gomega.HaveOccurred(), explain...)
}