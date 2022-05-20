// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentdextes

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	"github.com/verrazzano/verrazzano/tests/e2e/update/fluentd"
)

var (
	t                = framework.NewTestFramework("update fluentd external opensearch")
	extOpensearchURL string
	extOpensearchSec string
)

var _ = t.BeforeSuite(func() {
	cr := update.GetCR()
	fluentd := cr.Spec.Components.Fluentd
	extOpensearchURL = fluentd.ElasticsearchURL
	extOpensearchSec = fluentd.ElasticsearchSecret
})

var _ = t.AfterSuite(func() {
	fluentd.ValidateDaemonset(extOpensearchURL, extOpensearchSec, "")
})

var _ = t.Describe("Update Fluentd", Label("f:platform-lcm.update"), func() {
	t.Describe("Update to default Opensearch", Label("f:platform-lcm.fluentd-default-opensearch"), func() {
		t.It("default Opensearch", func() {
			m := &fluentd.FluentdModifier{Component: vzapi.FluentdComponent{}}
			fluentd.ValidateUpdate(m, "")
			fluentd.ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "")
		})
	})
	t.Describe("Update to external Opensearch", Label("f:platform-lcm.fluentd-external-opensearch"), func() {
		t.It("external Opensearch", func() {
			pkg.Log(pkg.Info, fmt.Sprintf("Update fluentd to use %v and %v", extOpensearchURL, extOpensearchSec))
			m := &fluentd.FluentdModifier{Component: vzapi.FluentdComponent{
				ElasticsearchSecret: extOpensearchSec,
				ElasticsearchURL:    extOpensearchURL,
			}}
			fluentd.ValidateUpdate(m, "")
			fluentd.ValidateDaemonset(extOpensearchURL, extOpensearchSec, "")
		})
	})
})
