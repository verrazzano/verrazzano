// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package alacarte

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
)

const (
	waitTimeout     = 10 * time.Minute
	pollingInterval = 5 * time.Second
)

const (
	promEdgeStack          = "prom-edge-stack"
	appStack               = "app-stack"
	istioAppStack          = "istio-app-stack"
	clusterManagementStack = "cluster-management-stack"
	noneProfile            = "none-profile"
)

var (
	t        = framework.NewTestFramework("a_la_carte")
	trueVal  = true
	falseVal = false
)

var failed = false
var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

type prometheusEdgeStackModifier struct {
}

type appStackModifier struct {
}

type istioAppStackModifier struct {
}

type clusterManagementStackModifier struct {
}

type noneModifier struct {
}

func (m prometheusEdgeStackModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.PrometheusOperator = &vzapi.PrometheusOperatorComponent{Enabled: &trueVal}
	cr.Spec.Components.Prometheus = &vzapi.PrometheusComponent{Enabled: &trueVal}
	cr.Spec.Components.PrometheusNodeExporter = &vzapi.PrometheusNodeExporterComponent{Enabled: &trueVal}
	cr.Spec.Components.PrometheusAdapter = &vzapi.PrometheusAdapterComponent{Enabled: &trueVal}
	cr.Spec.Components.PrometheusPushgateway = &vzapi.PrometheusPushgatewayComponent{Enabled: &trueVal}
	cr.Spec.Components.KubeStateMetrics = &vzapi.KubeStateMetricsComponent{Enabled: &trueVal}
}

func (m appStackModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.PrometheusNodeExporter = &vzapi.PrometheusNodeExporterComponent{Enabled: &falseVal}
	cr.Spec.Components.PrometheusAdapter = &vzapi.PrometheusAdapterComponent{Enabled: &falseVal}
	cr.Spec.Components.PrometheusPushgateway = &vzapi.PrometheusPushgatewayComponent{Enabled: &falseVal}
	cr.Spec.Components.KubeStateMetrics = &vzapi.KubeStateMetricsComponent{Enabled: &falseVal}

	cr.Spec.Components.ApplicationOperator = &vzapi.ApplicationOperatorComponent{Enabled: &trueVal}
	cr.Spec.Components.AuthProxy = &vzapi.AuthProxyComponent{Enabled: &trueVal}
	cr.Spec.Components.CertManager = &vzapi.CertManagerComponent{Enabled: &falseVal}
	cr.Spec.Components.ClusterIssuer = &vzapi.ClusterIssuerComponent{Enabled: &trueVal}
	cr.Spec.Components.Fluentd = &vzapi.FluentdComponent{Enabled: &trueVal}
	cr.Spec.Components.Grafana = &vzapi.GrafanaComponent{Enabled: &trueVal}
	cr.Spec.Components.Ingress = &vzapi.IngressNginxComponent{Enabled: &trueVal}
	cr.Spec.Components.Keycloak = &vzapi.KeycloakComponent{Enabled: &trueVal}
	cr.Spec.Components.MySQLOperator = &vzapi.MySQLOperatorComponent{Enabled: &trueVal}
	cr.Spec.Components.Elasticsearch = &vzapi.ElasticsearchComponent{Enabled: &trueVal}
	cr.Spec.Components.Kibana = &vzapi.KibanaComponent{Enabled: &trueVal}
	cr.Spec.Components.OAM = &vzapi.OAMComponent{Enabled: &trueVal}
	cr.Spec.Components.Verrazzano = &vzapi.VerrazzanoComponent{Enabled: &trueVal}
	cr.Spec.Components.JaegerOperator = &vzapi.JaegerOperatorComponent{Enabled: &trueVal}
}

func (m istioAppStackModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.Istio = &vzapi.IstioComponent{Enabled: &trueVal}
	cr.Spec.Components.Kiali = &vzapi.KialiComponent{Enabled: &trueVal}
}

func (m clusterManagementStackModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.ArgoCD = &vzapi.ArgoCDComponent{Enabled: &trueVal}
	cr.Spec.Components.CoherenceOperator = &vzapi.CoherenceOperatorComponent{Enabled: &trueVal}
	cr.Spec.Components.ClusterOperator = &vzapi.ClusterOperatorComponent{Enabled: &trueVal}
	cr.Spec.Components.Console = &vzapi.ConsoleComponent{Enabled: &trueVal}
	cr.Spec.Components.Rancher = &vzapi.RancherComponent{Enabled: &trueVal}
	cr.Spec.Components.RancherBackup = &vzapi.RancherBackupComponent{Enabled: &trueVal}
	cr.Spec.Components.Velero = &vzapi.VeleroComponent{Enabled: &trueVal}
	cr.Spec.Components.WebLogicOperator = &vzapi.WebLogicOperatorComponent{Enabled: &trueVal}
}

func (m noneModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.PrometheusOperator = &vzapi.PrometheusOperatorComponent{Enabled: &falseVal}
	cr.Spec.Components.Prometheus = &vzapi.PrometheusComponent{Enabled: &falseVal}
}

var beforeSuite = t.BeforeSuiteFunc(func() {
	Expect(getModifer(updateType)).ToNot(BeNil(), fmt.Sprintf("Provided update type %s was not a supported update type", updateType))
})

var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("Updating a la carte configuration", func() {
	t.It(fmt.Sprintf("to the %s and waiting for Verrazzano to become ready", updateType), func() {
		modifier := getModifer(updateType)
		if modifier == nil {
			AbortSuite(fmt.Sprintf("Unsupported modifier %s", updateType))
		}
		update.UpdateCRWithRetries(modifier, pollingInterval, waitTimeout)
	})
})

func getModifer(updateType string) update.CRModifier {
	switch updateType {
	case promEdgeStack:
		return prometheusEdgeStackModifier{}
	case appStack:
		return appStackModifier{}
	case istioAppStack:
		return istioAppStackModifier{}
	case clusterManagementStack:
		return clusterManagementStackModifier{}
	case noneProfile:
		return noneModifier{}
	default:
		return nil
	}
}
