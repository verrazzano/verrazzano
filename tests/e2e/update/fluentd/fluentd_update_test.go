// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentd

import (
	"fmt"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	pcons "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"time"
)

const (
	labelValidation      = "f:platform-lcm.fluentd-update-validation"
	opensearchURL        = "https://opensearch.example.com:9200"
	opensearchURLV1beta1 = "https://opensearch.v1beta1.example.com:9200"
)

var (
	t        = framework.NewTestFramework("update fluentd")
	tempuuid = uuid.NewString()[:7]
	extEsSec = "my-extsec-" + tempuuid
	wrongSec = "wrong-sec-" + tempuuid
	ociLgSec = "my-ocilog-" + tempuuid
	sysLogID = "my-sysLog-" + tempuuid
	defLogID = "my-defLog-" + tempuuid
)

var _ = t.AfterSuite(func() {
	pkg.DeleteSecret(pcons.VerrazzanoInstallNamespace, extEsSec)
	pkg.DeleteSecret(pcons.VerrazzanoInstallNamespace, wrongSec)
	start := time.Now()
	gomega.Eventually(func() bool {
		return ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "")
	}, tenMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", pkg.VmiESURL, time.Since(start)))
})

var _ = t.Describe("Update Fluentd", Label("f:platform-lcm.update"), func() {
	t.Describe("fluentd verify", Label("f:platform-lcm.fluentd-verify"), func() {
		t.It("fluentd default config", func() {
			start := time.Now()
			gomega.Eventually(func() bool {
				return ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "")
			}, tenMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", pkg.VmiESURL, time.Since(start)))
		})
	})

	t.Describe("Validate external Opensearch config", Label(labelValidation), func() {
		t.It("secret validation", func() {
			m := &FluentdModifier{Component: vzapi.FluentdComponent{
				ElasticsearchSecret: extEsSec + "missing",
				ElasticsearchURL:    opensearchURL,
			}}
			expectedError := "must be created"
			gomega.Expect(ValidateUpdate(m, expectedError)).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", expectedError))
		})
	})

	t.Describe("Update external Opensearch", Label("f:platform-lcm.fluentd-external-opensearch"), func() {
		t.It("external Opensearch", func() {
			pkg.CreateCredentialsSecret(pcons.VerrazzanoInstallNamespace, extEsSec, "user", "pw", map[string]string{})
			v1alpha1Modifier := &FluentdModifier{Component: vzapi.FluentdComponent{
				ElasticsearchSecret: extEsSec,
				ElasticsearchURL:    opensearchURL,
			}}
			v1beta1Modifier := &FluentdModifierV1beta1{Component: v1beta1.FluentdComponent{
				OpenSearchSecret: extEsSec,
				OpenSearchURL:    opensearchURLV1beta1,
			}}

			// Update CR using v1alpha1 API client.
			gomega.Expect(ValidateUpdate(v1alpha1Modifier, "")).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", ""))
			start := time.Now()
			gomega.Eventually(func() bool {
				return ValidateDaemonset(opensearchURL, extEsSec, "")
			}, tenMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", opensearchURL, time.Since(start)))
			fmt.Sprintf("Fluentd took %v to update", time.Since(start))

			//Update CR using v1beta1 API client.
			gomega.Expect(ValidateUpdateV1beta1(v1beta1Modifier, "")).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", ""))

			start = time.Now()
			gomega.Eventually(func() bool {
				return ValidateDaemonsetV1beta1(opensearchURLV1beta1, extEsSec, "")
			}, tenMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", opensearchURLV1beta1, time.Since(start)))
		})
	})

	t.Describe("Validate OCI logging config", Label(labelValidation), func() {
		t.It("secret validation", func() {
			m := &FluentdModifier{Component: vzapi.FluentdComponent{
				OCI: &vzapi.OciLoggingConfiguration{APISecret: wrongSec},
			}}
			expectedError1 := "must be created"
			gomega.Expect(ValidateUpdate(m, expectedError1)).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", expectedError1))

			pkg.CreateCredentialsSecret(pcons.VerrazzanoInstallNamespace, wrongSec, "api", "pw", map[string]string{})

			expectedError2 := "Did not find OCI configuration"
			gomega.Expect(ValidateUpdate(m, expectedError2)).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", expectedError2))
		})
	})

	t.Describe("Update OCI logging", Label("f:platform-lcm.fluentd-oci-logging"), func() {
		t.It(" OCI logging", func() {
			createOciLoggingSecret(ociLgSec)
			m := &FluentdModifier{Component: vzapi.FluentdComponent{OCI: &vzapi.OciLoggingConfiguration{
				APISecret:       ociLgSec,
				SystemLogID:     sysLogID,
				DefaultAppLogID: defLogID,
			}}}
			gomega.Expect(ValidateUpdate(m, "")).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", ""))

			start := time.Now()
			gomega.Eventually(func() bool {
				return ValidateDaemonset("", "", ociLgSec)
			}, tenMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", "", time.Since(start)))

			gomega.Eventually(func() bool {
				return ValidateConfigMap(sysLogID, defLogID)
			}, oneMinute, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("ConfigMap %s is not ready", fluentdName+"-config"))

		})
	})

	t.Describe("Validate extra Volume Mounts", Label(labelValidation), func() {
		t.It("extraVolumeMounts validation", func() {
			m := &FluentdModifier{Component: vzapi.FluentdComponent{
				ExtraVolumeMounts: []vzapi.VolumeMount{{Source: "/var/log"}},
			}}
			expectedError := "duplicate mount path found"
			gomega.Expect(ValidateUpdate(m, expectedError)).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", expectedError))
		})
	})

	t.Describe("Update extraVolumeMounts", Label("f:platform-lcm.fluentd-extra-volume-mounts"), func() {
		t.It("extraVolumeMounts", func() {
			vm := vzapi.VolumeMount{Source: "/var/log", Destination: "/home/varLog"}
			m := &FluentdModifier{Component: vzapi.FluentdComponent{
				ExtraVolumeMounts: []vzapi.VolumeMount{vm},
			}}
			gomega.Expect(ValidateUpdate(m, "")).Should(gomega.BeTrue(), fmt.Sprintf("expected error %v", ""))

			start := time.Now()
			gomega.Eventually(func() bool {
				return ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "", vm)
			}, tenMinutes, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", pkg.VmiESURL, time.Since(start)))
		})
	})
})
