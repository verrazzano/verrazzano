package registerlabels

import (
	gv2 "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"time"
)

var (
	t = framework.NewTestFramework(" ")
)
var _ = t.BeforeSuite(func() {
	start := time.Now()
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})
var failed = false
var _ = t.AfterEach(func() {
	failed = failed || gv2.CurrentSpecReport().Failed()
})

var _ = t.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
	start := time.Now()
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe(" ", gv2.Label("f:platform-lcm.install",
	"f:platform-lcm.uninstall",
	"f:platform-lcm.private-registry",
	"f:platform-lcm.upgrade",
	"f:app-lcm.oam",
	"f:app-lcm.weblogic-workload",
	"f:app-lcm.spring-workload",
	"f:app-lcm.helidon-workload",
	"f:app-lcm.coherence-workload",
	"f:app-lcm.poko",
	"f:app-lcm.logging-trait",
	"f:app-lcm.gitops",
	"f:security.rbac",
	"f:security.netpol",
	"f:security.authpol",
	"f:security.apiproxy",
	"f:mesh.ingress",
	"f:mesh.traffic-mgmt",
	"f:multi-cluster.mc-app-lcm",
	"f:multi-cluster.register",
	"f:multi-cluster.deregister",
	"f:cert-mgmt",
	"f:dns-mgmt",
	"f:observability.logging.es",
	"f:observability.logging.kibana",
	"f:observability.logging.oci",
	"f:observability.monitoring.prom",
	"f:observability.monitoring.graf",
	"f:observability.monitoring.oci",
	"f:observability.monitoring.dist-trace",
	"f:ui.console",
	"f:ui.cli",
	"f:ui.api",
	"f:diag-tool",
	"f:infra-lcm",
	"f:oci-integration.logging",
	"f:oci-integration.monitoring",
	"f:oci-integration.apm",
	"f:oci-integration.osok",
	"f:oci-integration.mesh",
	"f:oci-integration.devops",
	"f:oci-integration.rm"), func() {})
