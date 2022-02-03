package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"k8s.io/klog/v2"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"

	//appsv1 "k8s.io/api/apps/v1"
	//v1 "k8s.io/api/core/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/util/wait"
	//"k8s.io/component-base/logs"
	//"k8s.io/component-base/version"
	//commontest "k8s.io/kubernetes/test/e2e/common"
	//"k8s.io/kubernetes/test/e2e/framework"
	//"k8s.io/kubernetes/test/e2e/framework/daemonset"
	//e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
	//e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	//e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	//e2ereporters "k8s.io/kubernetes/test/e2e/reporters"
	//utilnet "k8s.io/utils/net"

	//clientset "k8s.io/client-go/kubernetes"
	//// ensure auth plugins are loaded
	//_ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	//// ensure that cloud providers are loaded
	//_ "k8s.io/kubernetes/test/e2e/framework/providers/aws"
	//_ "k8s.io/kubernetes/test/e2e/framework/providers/azure"
	//_ "k8s.io/kubernetes/test/e2e/framework/providers/gce"
	//_ "k8s.io/kubernetes/test/e2e/framework/providers/kubemark"
	//_ "k8s.io/kubernetes/test/e2e/framework/providers/openstack"
	//_ "k8s.io/kubernetes/test/e2e/framework/providers/vsphere"
	//
	//// Ensure that logging flags are part of the command line.
	//_ "k8s.io/component-base/logs/testinit"
)

const (
	// namespaceCleanupTimeout is how long to wait for the namespace to be deleted.
	// If there are any orphaned namespaces to clean up, this test is running
	// on a long lived cluster. A long wait here is preferably to spurious test
	// failures caused by leaked resources from a previous test run.
	namespaceCleanupTimeout = 15 * time.Minute
)

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	// Reference common test to make the import valid.
	commontest.CurrentSuite = commontest.E2E
	setupSuite()
	return nil
}, func(data []byte) {
	// Run on all Ginkgo nodes
	setupSuitePerGinkgoNode()
})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	CleanupSuite()
}, func() {
	AfterSuiteActions()
})

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
// If a "report directory" is specified, one or more JUnit test reports will be
// generated in this directory, and cluster logs will also be saved.
// This function is called on each Ginkgo node in parallel mode.
func RunE2ETests(t *testing.T) {
	logs.InitLogs()
	defer logs.FlushLogs()

	gomega.RegisterFailHandler(framework.Fail)
	// Disable skipped tests unless they are explicitly requested.
	if config.GinkgoConfig.FocusString == "" && config.GinkgoConfig.SkipString == "" {
		config.GinkgoConfig.SkipString = `\[Flaky\]|\[Feature:.+\]`
	}

	// Run tests through the Ginkgo runner with output to console + JUnit for Jenkins
	var r []ginkgo.Reporter
	if framework.TestContext.ReportDir != "" {
		// TODO: we should probably only be trying to create this directory once
		// rather than once-per-Ginkgo-node.
		if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
			klog.Errorf("Failed creating report directory: %v", err)
		} else {
			r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode))))
		}
	}

	// Stream the progress to stdout and optionally a URL accepting progress updates.
	r = append(r, e2ereporters.NewProgressReporter(framework.TestContext.ProgressReportURL))

	// The DetailsRepoerter will output details about every test (name, files, lines, etc) which helps
	// when documenting our tests.
	if len(framework.TestContext.SpecSummaryOutput) > 0 {
		r = append(r, e2ereporters.NewDetailsReporterFile(framework.TestContext.SpecSummaryOutput))
	}

	klog.Infof("Starting e2e run %q on Ginkgo node %d", framework.RunID, config.GinkgoConfig.ParallelNode)
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "Kubernetes e2e suite", r)
}